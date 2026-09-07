package statement

import (
	"errors"
	"strconv"
	"strings"
)

// ParseMinor converts a decimal amount string into minor units (centavos).
//
// Providers send money as a STRING — Pluggy returns "40.00", "-1018.18" — and the
// obvious route is strconv.ParseFloat followed by a multiply. That reintroduces
// exactly the precision problem the integer minor units were adopted to remove: it is
// the same mistake as float64 arithmetic, hidden inside a parser.
//
// So the string is split on the decimal point and read as integers. No floating point
// touches the value.
func ParseMinor(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty amount")
	}

	negative := false
	switch s[0] {
	case '-':
		negative, s = true, s[1:]
	case '+':
		s = s[1:]
	}

	whole, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		whole, frac = s[:i], s[i+1:]
	}
	if whole == "" {
		whole = "0"
	}

	// Two decimal places, padded or refused. Silently truncating a third digit would
	// lose a centavo per line and call it rounding.
	switch len(frac) {
	case 0:
		frac = "00"
	case 1:
		frac += "0"
	case 2:
	default:
		return 0, errors.New("amount has more than two decimal places: " + s)
	}

	wholeUnits, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, errors.New("malformed amount: " + s)
	}
	fracUnits, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, errors.New("malformed amount: " + s)
	}

	minor := wholeUnits*100 + fracUnits
	if negative {
		minor = -minor
	}
	return minor, nil
}
