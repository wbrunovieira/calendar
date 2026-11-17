package usecases

import (
	"testing"
	"time"
)

// A purchase dated TODAY is not historical, at any hour of the day.
//
// `time.Now().Truncate(24 * time.Hour)` snaps to a UTC midnight, not to the
// local one. The server runs in America/Sao_Paulo (-03), so from 21:00 local
// the truncated "today" is already TOMORROW in UTC and a transaction dated today
// compares as being in the past. Historical transactions skip balance and
// credit-limit validation, so those guards were off for three hours every
// evening — confirmed by running the old expression across the day.
//
// This is why three tests in this package failed only in the evening.
func TestIsHistoricalDate(t *testing.T) {
	// No t.Skip here. A machine without tzdata would make the whole regression
	// disappear silently, which is the failure this test exists to prevent;
	// issuerLocation falls back to a fixed -03:00 for the same reason.
	brt := issuerLocation
	dia := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("data de teste invalida %q: %v", s, err)
		}
		return d
	}

	casos := []struct {
		nome    string
		data    time.Time
		agora   time.Time
		passado bool
	}{
		{"hoje, de manha", dia("2026-11-16"), time.Date(2026, 11, 16, 9, 0, 0, 0, brt), false},
		{"hoje, 20h59", dia("2026-11-16"), time.Date(2026, 11, 16, 20, 59, 0, 0, brt), false},
		// A janela do bug: 21h BRT ja e meia-noite UTC.
		{"hoje, 21h", dia("2026-11-16"), time.Date(2026, 11, 16, 21, 0, 0, 0, brt), false},
		{"hoje, 23h59", dia("2026-11-16"), time.Date(2026, 11, 16, 23, 59, 0, 0, brt), false},
		{"ontem", dia("2026-11-15"), time.Date(2026, 11, 16, 9, 0, 0, 0, brt), true},
		{"ontem, 22h", dia("2026-11-15"), time.Date(2026, 11, 16, 22, 0, 0, 0, brt), true},
		{"amanha", dia("2026-11-17"), time.Date(2026, 11, 16, 9, 0, 0, 0, brt), false},
		{"amanha, 22h", dia("2026-11-17"), time.Date(2026, 11, 16, 22, 0, 0, 0, brt), false},
	}

	for _, c := range casos {
		if got := isHistoricalDate(c.data, c.agora); got != c.passado {
			t.Errorf("%s: isHistoricalDate = %v, want %v", c.nome, got, c.passado)
		}
	}
}

// The real production split: the SERVER runs in UTC (the calendar-finances
// container has no TZ) and the dates come from a browser in Brazil. The first
// version of this fix compared calendar days in the server's own zone, so both
// sides of every test moved together and the table stayed green while the guard
// was off in production.
//
// This pins the two clocks independently: it asks the question as the server
// would at 01:36 UTC — which is 22:36 the PREVIOUS day in Sao Paulo — about a
// purchase the user dated with their own calendar.
func TestADateFromBrazilIsNotHistoricalJustBecauseTheServerRunsInUTC(t *testing.T) {
	compraDeHoje, err := time.Parse("2006-01-02", "2026-11-16")
	if err != nil {
		t.Fatalf("data invalida: %v", err)
	}
	// 2026-11-17 01:36 UTC == 2026-11-16 22:36 em Sao Paulo.
	servidorEmUTC := time.Date(2026, 11, 17, 1, 36, 0, 0, time.UTC)

	if isHistoricalDate(compraDeHoje, servidorEmUTC) {
		t.Error("purchase dated today read as historical because the server clock is in UTC — the guards are skipped")
	}
}

// And the day really does roll over, in Sao Paulo time.
func TestTheDayRollsOverAtMidnightInSaoPaulo(t *testing.T) {
	ontem, _ := time.Parse("2006-01-02", "2026-11-16")
	// 2026-11-17 03:30 UTC == 00:30 do dia 17 em Sao Paulo: ja virou.
	depoisDaMeiaNoiteBRT := time.Date(2026, 11, 17, 3, 30, 0, 0, time.UTC)

	if !isHistoricalDate(ontem, depoisDaMeiaNoiteBRT) {
		t.Error("16/11 should be historical at 00:30 BRT on 17/11")
	}
}

// clientDay is the calendar day as the USER sees it — São Paulo — which is what
// a browser in Brazil sends as `occurredOn`.
//
// Tests used to build these with `time.Now().Format(...)`, i.e. the SERVER's
// clock. Both sides of every date comparison then moved together with TZ, which
// is precisely why a timezone bug that was live in production could not make a
// single test go red.
func clientDay(offset int) string {
	return time.Now().In(issuerLocation).AddDate(0, 0, offset).Format("2006-01-02")
}
