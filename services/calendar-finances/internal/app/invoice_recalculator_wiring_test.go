package app

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// A use case that knows how to refresh a bill's total but is never handed a
// recalculator is a no-op that reads like a feature. That is not hypothetical: the
// whole defect this guard exists for was one unwired collaborator, and reverting the
// wiring left every test in the repository green, because nothing asserts on how
// New() hooks things together.
//
// This is a source-level gate for the same reason the status-transition gate in
// transaction.go is one: the use cases are locals inside New(), unreachable from a
// test, and the thing worth pinning is not one particular call but the RULE — every
// use case that can refresh a bill is given something to refresh it with. A new one
// added next year is covered without anyone remembering this file exists.
func TestEveryInvoiceRecalculatorAwareUseCaseIsWired(t *testing.T) {
	useCaseDir := filepath.Join("..", "application", "usecases")
	entries, err := os.ReadDir(useCaseDir)
	if err != nil {
		t.Fatalf("reading the use case package: %v", err)
	}

	declares := regexp.MustCompile(`func \(uc \*(\w+)\) SetInvoiceRecalculator\(`)
	aware := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || len(name) > 8 && name[len(name)-8:] == "_test.go" {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(useCaseDir, name))
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		for _, m := range declares.FindAllSubmatch(src, -1) {
			aware[string(m[1])] = true
		}
	}

	if len(aware) == 0 {
		t.Fatal("found no use case declaring SetInvoiceRecalculator — this guard has stopped guarding anything")
	}

	wiring, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("reading app.go: %v", err)
	}

	// Map each local in New() to the use case type it was constructed from, then
	// check that same local is handed a recalculator.
	constructs := regexp.MustCompile(`(\w+)\s*:?=\s*usecases\.New(\w+)\(`)
	builtFrom := map[string][]string{} // use case type -> locals holding one
	for _, m := range constructs.FindAllSubmatch(wiring, -1) {
		local, typ := string(m[1]), string(m[2])
		builtFrom[typ] = append(builtFrom[typ], local)
	}

	// The argument is checked, not just the call. Wiring a literal nil satisfies
	// "is it called?" while leaving the refresh exactly as dead as never calling it.
	wired := regexp.MustCompile(`(\w+)\.SetInvoiceRecalculator\(\s*(\w+)\s*\)`)
	isWired := map[string]bool{}
	for _, m := range wired.FindAllSubmatch(wiring, -1) {
		if string(m[2]) == "nil" {
			continue
		}
		isWired[string(m[1])] = true
	}

	var missing []string
	for typ := range aware {
		locals, ok := builtFrom[typ]
		if !ok {
			// Not constructed in New() at all: nothing to wire, and nothing to
			// claim. Constructed elsewhere it is that caller's responsibility.
			continue
		}
		found := false
		for _, l := range locals {
			if isWired[l] {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, typ)
		}
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("these use cases can refresh a bill's total but New() never gives them a recalculator, "+
			"so the refresh silently does nothing: %v", missing)
	}
}
