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
	brt, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skip("no tzdata in this environment")
	}
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
