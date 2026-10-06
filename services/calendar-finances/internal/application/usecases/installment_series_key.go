package usecases

import (
	"regexp"
	"strconv"
	"strings"
)

// installmentSeriesName returns a name shared by every part of one instalment plan.
//
// Parts have to be grouped before they can be counted, and the raw description does
// not group them: people write the instalment number INTO it. "Gomez Studio - parcela
// 1/5" and "... 2/5" are two strings, so a healthy five-part plan became five series
// of one part each and raised five alarms. Three plans produced eleven false alarms
// between them, which buried the real gap -- "The Dark Film - loja virtual" really is
// missing its 2/3.
//
// Only the row's OWN number over its OWN total is removed. That matters: one plan is
// called "Parcelamento fatura 02/09 - parcela 1/3", and stripping any N/M would have
// eaten the 02/09 that names it.
func installmentSeriesName(description string, number, total int) string {
	name := strings.TrimSpace(description)
	if number > 0 && total > 1 {
		name = installmentFraction(number, total).ReplaceAllString(name, " ")
	}
	// One plan, one name, however it was capitalised or spaced.
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// fractionCache keeps the compiled patterns, since the check walks every transaction
// of every account.
var fractionCache = map[[2]int]*regexp.Regexp{}

func installmentFraction(number, total int) *regexp.Regexp {
	k := [2]int{number, total}
	if re, ok := fractionCache[k]; ok {
		return re
	}
	n, t := strconv.Itoa(number), strconv.Itoa(total)
	// An optional separator, an optional word for "instalment", and the fraction,
	// with or without parentheses. Anchored on the digits so nothing else is touched.
	re := regexp.MustCompile(`(?i)[\s\-–—]*\(?\s*(?:parcelas?|parc\.?|installments?|inst\.?)?\s*` +
		regexp.QuoteMeta(n) + `\s*/\s*` + regexp.QuoteMeta(t) + `\s*\)?`)
	fractionCache[k] = re
	return re
}
