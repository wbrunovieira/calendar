//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	httpHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/http/middleware"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

const syncSecret = "e2e-shared-secret"

// contractRouter wires the route the way app.go does, guard included. The guard is
// part of the route under test: a sync endpoint that works but is reachable without
// the secret is not the thing we are shipping.
func contractRouter(t *testing.T, db *sql.DB, profileID string) *mux.Router {
	t.Helper()

	uc := usecases.NewSyncContractUseCase(
		persistence.NewContractRepository(db),
		persistence.NewCostCenterRepository(db),
		profileID,
	)
	handler := httpHandlers.NewContractHandlers(uc)

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Handle("/contracts/sync",
		middleware.RequireWebhookSecret(syncSecret, http.HandlerFunc(handler.Sync))).Methods("POST")
	return r
}

func postSync(t *testing.T, r *mux.Router, secret, body string) (int, map[string]any) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/contracts/sync", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set(middleware.WebhookSecretHeader, secret)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var parsed map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	return rec.Code, parsed
}

func seedSyncProfile(t *testing.T, db *sql.DB) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`
		INSERT INTO finance.profiles (calendar_id, name, type)
		VALUES ('e2e-' || gen_random_uuid(), 'E2E CRM', 'BUSINESS') RETURNING id
	`).Scan(&id); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.contracts WHERE profile_id = $1`, id)
		db.Exec(`DELETE FROM finance.cost_centers WHERE profile_id = $1`, id)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, id)
	})
	return id
}

func payload(dealID, status, updatedAt string, total string) string {
	return fmt.Sprintf(`{
		"source":"wb-crm",
		"deal":{"id":%q,"title":"Site institucional","totalValue":%s,
		        "currency":"BRL","status":%q,"closedAt":null,"updatedAt":%q},
		"organization":{"id":"org-e2e-1","name":"Refrigeracao Garrido"}
	}`, dealID, total, status, updatedAt)
}

func outcome(t *testing.T, body map[string]any) string {
	t.Helper()
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data in response: %v", body)
	}
	return data["outcome"].(string)
}

func TestContractSyncEndToEnd(t *testing.T) {
	db := testDB(t)
	profileID := seedSyncProfile(t, db)
	r := contractRouter(t, db, profileID)

	t.Run("refuses an unauthenticated caller", func(t *testing.T) {
		code, _ := postSync(t, r, "", payload("deal-e2e-1", "open", "2026-09-23T12:00:00Z", "350.00"))
		if code != http.StatusUnauthorized {
			t.Fatalf("code = %d, want 401", code)
		}
		var n int
		db.QueryRow(`SELECT count(*) FROM finance.contracts WHERE profile_id = $1`, profileID).Scan(&n)
		if n != 0 {
			t.Fatal("an unauthenticated call wrote a contract")
		}
	})

	t.Run("first delivery creates the client and the contract", func(t *testing.T) {
		code, body := postSync(t, r, syncSecret, payload("deal-e2e-1", "open", "2026-09-23T12:00:00Z", "350.00"))
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		if got := outcome(t, body); got != "created" {
			t.Fatalf("outcome = %q, want created", got)
		}

		var total int64
		var status, centerName string
		err := db.QueryRow(`
			SELECT c.total_minor, c.status, cc.name
			FROM finance.contracts c
			JOIN finance.cost_centers cc ON cc.id = c.cost_center_id
			WHERE c.source = 'wb-crm' AND c.external_id = 'deal-e2e-1'`).Scan(&total, &status, &centerName)
		if err != nil {
			t.Fatalf("reading back what was stored: %v", err)
		}
		if total != 35000 || status != "OPEN" || centerName != "Refrigeracao Garrido" {
			t.Fatalf("stored total=%d status=%s client=%s", total, status, centerName)
		}
	})

	t.Run("a redelivery writes nothing", func(t *testing.T) {
		code, body := postSync(t, r, syncSecret, payload("deal-e2e-1", "open", "2026-09-23T12:00:00Z", "350.00"))
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		if got := outcome(t, body); got != "stale" {
			t.Fatalf("outcome = %q, want stale", got)
		}
		var n int
		db.QueryRow(`SELECT count(*) FROM finance.contracts WHERE source='wb-crm' AND external_id='deal-e2e-1'`).Scan(&n)
		if n != 1 {
			t.Fatalf("%d rows; the redelivery duplicated the contract", n)
		}
	})

	t.Run("a won deal updates in place and reuses the client", func(t *testing.T) {
		code, body := postSync(t, r, syncSecret, payload("deal-e2e-1", "won", "2026-09-23T15:00:00Z", "420.00"))
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		if got := outcome(t, body); got != "updated" {
			t.Fatalf("outcome = %q, want updated", got)
		}

		var total int64
		var status string
		db.QueryRow(`SELECT total_minor, status FROM finance.contracts
			WHERE source='wb-crm' AND external_id='deal-e2e-1'`).Scan(&total, &status)
		if total != 42000 || status != "WON" {
			t.Fatalf("stored total=%d status=%s", total, status)
		}

		var centers int
		db.QueryRow(`SELECT count(*) FROM finance.cost_centers WHERE profile_id = $1`, profileID).Scan(&centers)
		if centers != 1 {
			t.Fatalf("%d cost centers; the same organization produced a twin", centers)
		}
	})

	t.Run("an out-of-order delivery does not undo the win", func(t *testing.T) {
		code, body := postSync(t, r, syncSecret, payload("deal-e2e-1", "open", "2026-09-23T13:00:00Z", "350.00"))
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		if got := outcome(t, body); got != "stale" {
			t.Fatalf("outcome = %q, want stale", got)
		}
		var status string
		db.QueryRow(`SELECT status FROM finance.contracts
			WHERE source='wb-crm' AND external_id='deal-e2e-1'`).Scan(&status)
		if status != "WON" {
			t.Fatalf("status = %s; an older delivery overwrote newer state", status)
		}
	})

	t.Run("a lost deal is accepted", func(t *testing.T) {
		code, body := postSync(t, r, syncSecret, payload("deal-e2e-1", "lost", "2026-09-23T18:00:00Z", "420.00"))
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		var status string
		db.QueryRow(`SELECT status FROM finance.contracts
			WHERE source='wb-crm' AND external_id='deal-e2e-1'`).Scan(&status)
		if status != "LOST" {
			t.Fatalf("status = %s, want LOST", status)
		}
	})

	t.Run("a payload the sender got wrong answers 400, not 500", func(t *testing.T) {
		for name, body := range map[string]string{
			"unknown status": payload("deal-e2e-2", "abandoned", "2026-09-23T12:00:00Z", "10.00"),
			"no updatedAt":   payload("deal-e2e-2", "open", "0001-01-01T00:00:00Z", "10.00"),
			"negative value": payload("deal-e2e-2", "open", "2026-09-23T12:00:00Z", "-1"),
		} {
			t.Run(name, func(t *testing.T) {
				code, _ := postSync(t, r, syncSecret, body)
				if code != http.StatusBadRequest {
					t.Fatalf("code = %d, want 400", code)
				}
			})
		}
	})

	t.Run("an offset-bearing stamp still orders correctly", func(t *testing.T) {
		// The CRM emits ISO strings; whether they carry Z or a local offset is a
		// formatting choice on its side, and the ledger must order by INSTANT either
		// way. A plain TIMESTAMP column drops the offset, so "15:00-03:00" stored as
		// 15:00 and read back as 15:00Z — three hours early — and a genuinely older
		// delivery then beat a newer one. This is that sequence, end to end.
		win := payload("deal-e2e-tz", "won", "2026-09-23T15:00:00-03:00", "420.00") // 18:00Z
		if code, body := postSync(t, r, syncSecret, win); code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}

		older := payload("deal-e2e-tz", "open", "2026-09-23T17:00:00Z", "350.00") // an hour BEFORE
		code, body := postSync(t, r, syncSecret, older)
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		if got := outcome(t, body); got != "stale" {
			t.Fatalf("outcome = %q, want stale", got)
		}

		var status string
		var total int64
		db.QueryRow(`SELECT status, total_minor FROM finance.contracts
			WHERE source='wb-crm' AND external_id='deal-e2e-tz'`).Scan(&status, &total)
		if status != "WON" || total != 42000 {
			t.Fatalf("status=%s total=%d; an older delivery undid the win", status, total)
		}
	})

	t.Run("a redelivery with an offset stamp is still recognised as stale", func(t *testing.T) {
		// The other half of the same defect: with the instant shifted, the stored
		// value is always behind the incoming one, so nothing is EVER stale and the
		// endpoint rewrites the row on every single delivery.
		same := payload("deal-e2e-tz2", "open", "2026-09-23T09:00:00-03:00", "100.00")
		if code, _ := postSync(t, r, syncSecret, same); code != http.StatusOK {
			t.Fatal("first delivery failed")
		}
		for i := 0; i < 2; i++ {
			code, body := postSync(t, r, syncSecret, same)
			if code != http.StatusOK {
				t.Fatalf("code = %d", code)
			}
			if got := outcome(t, body); got != "stale" {
				t.Fatalf("redelivery %d: outcome = %q, want stale", i+1, got)
			}
		}
	})

	t.Run("a currency the column cannot hold is 400, not 500", func(t *testing.T) {
		body := `{"source":"wb-crm",
			"deal":{"id":"deal-e2e-ccy","title":"Proposta","totalValue":null,
			        "currency":"DOLLAR","status":"open","closedAt":null,
			        "updatedAt":"2026-09-23T12:00:00Z"},
			"organization":{"id":"org-e2e-1","name":"Refrigeracao Garrido"}}`
		code, _ := postSync(t, r, syncSecret, body)
		if code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400 — 500 tells the sender to retry a payload that can never work", code)
		}
	})

	t.Run("a deal with no value yet is stored without inventing a zero", func(t *testing.T) {
		code, body := postSync(t, r, syncSecret, payload("deal-e2e-3", "open", "2026-09-23T12:00:00Z", "null"))
		if code != http.StatusOK {
			t.Fatalf("code = %d body = %v", code, body)
		}
		var total sql.NullInt64
		db.QueryRow(`SELECT total_minor FROM finance.contracts
			WHERE source='wb-crm' AND external_id='deal-e2e-3'`).Scan(&total)
		if total.Valid {
			t.Fatalf("total_minor = %d, want NULL", total.Int64)
		}
	})
}
