package app

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
// Setters are DISCOVERED, not listed. An earlier version carried a hardcoded slice
// of two names while claiming to cover "every collaborator attached after
// construction" — which made the guard itself an instance of the thing it guards
// against: cheap to add, cheap to forget, and silent when forgotten. The package
// declares four such setters today; a fifth added next year is covered without
// anyone editing this file.
func TestEveryWiredCollaboratorIsAttached(t *testing.T) {
	setters := discoverSetters(t)
	if len(setters) < 2 {
		t.Fatalf("found %d setters — this guard has stopped guarding anything", len(setters))
	}
	for _, setter := range setters {
		t.Run(setter, func(t *testing.T) { assertSettersAreWired(t, setter) })
	}
}

// discoverSetters returns every Set* method the use case package declares on a use
// case, deduplicated and ordered.
func discoverSetters(t *testing.T) []string {
	t.Helper()
	found := map[string]bool{}
	decl := regexp.MustCompile(`func \(\w+ \*\w+\) (Set\w+)\(`)
	for _, src := range useCaseSources(t) {
		for _, m := range decl.FindAllSubmatch(src, -1) {
			found[string(m[1])] = true
		}
	}
	names := make([]string, 0, len(found))
	for n := range found {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func useCaseSources(t *testing.T) [][]byte {
	t.Helper()
	dir := filepath.Join("..", "application", "usecases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the use case package: %v", err)
	}
	var out [][]byte
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		out = append(out, src)
	}
	return out
}

func assertSettersAreWired(t *testing.T, setter string) {
	t.Helper()
	wiring, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("reading app.go: %v", err)
	}
	missing := unwiredUseCases(useCaseSources(t), wiring, setter)
	if len(missing) > 0 {
		t.Fatalf("these use cases declare %s but New() never calls it with a real "+
			"collaborator, so it silently does nothing: %v", setter, missing)
	}
}

// unwiredUseCases is the whole rule, as a function of its inputs so it can be
// exercised on synthetic sources. The version before this took its content straight
// off disk, which left its own logic — in particular the refusal of a literal nil —
// provable only by breaking the real app.go, and therefore untested.
func unwiredUseCases(useCaseSources [][]byte, wiring []byte, setter string) []string {
	declares := regexp.MustCompile(`func \(\w+ \*(\w+)\) ` + regexp.QuoteMeta(setter) + `\(`)
	aware := map[string]bool{}
	for _, src := range useCaseSources {
		for _, m := range declares.FindAllSubmatch(src, -1) {
			aware[string(m[1])] = true
		}
	}

	// Map each local in New() to the use case type it was constructed from.
	constructs := regexp.MustCompile(`(\w+)\s*:?=\s*usecases\.New(\w+)\(`)
	builtFrom := map[string][]string{}
	for _, m := range constructs.FindAllSubmatch(wiring, -1) {
		builtFrom[string(m[2])] = append(builtFrom[string(m[2])], string(m[1]))
	}

	// The argument is checked, not just the call: wiring a literal nil satisfies
	// "is it called?" while leaving the collaborator exactly as dead as never
	// calling it. Anything that is not nil counts, because a real wiring may pass a
	// composite literal or a constructor call rather than a bare identifier —
	// matching only identifiers reported three correctly-wired SetUnitOfWork call
	// sites as missing.
	wired := regexp.MustCompile(`(\w+)\.` + regexp.QuoteMeta(setter) + `\(\s*([^)]*?)\s*\)`)
	isWired := map[string]bool{}
	for _, m := range wired.FindAllSubmatch(wiring, -1) {
		if arg := strings.TrimSpace(string(m[2])); arg == "" || arg == "nil" {
			continue
		}
		isWired[string(m[1])] = true
	}

	var missing []string
	for typ := range aware {
		locals, ok := builtFrom[typ]
		if !ok {
			// Not constructed in New() at all: nothing to wire, and nothing to
			// claim. Constructed elsewhere, it is that caller's responsibility.
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
	return missing
}

// The guard's own rules, on sources written for the purpose. Without these the
// refusal of a literal nil was provable only by vandalising app.go.
func TestUnwiredUseCases(t *testing.T) {
	const decl = "func (uc *FooUseCase) SetBar(r Bar) {}\n"

	cases := []struct {
		name    string
		wiring  string
		missing []string
	}{
		{"wired with a real collaborator",
			"foo := usecases.NewFooUseCase(db)\nfoo.SetBar(barRepo)\n", nil},
		{"wired with a composite literal",
			"foo := usecases.NewFooUseCase(db)\nfoo.SetBar(&boundBar{x: 1})\n", nil},
		{"wired with a constructor call",
			"foo := usecases.NewFooUseCase(db)\nfoo.SetBar(persistence.NewBar(db))\n", nil},
		{"never called",
			"foo := usecases.NewFooUseCase(db)\n", []string{"FooUseCase"}},
		{"called with a literal nil",
			"foo := usecases.NewFooUseCase(db)\nfoo.SetBar(nil)\n", []string{"FooUseCase"}},
		{"called with nothing at all",
			"foo := usecases.NewFooUseCase(db)\nfoo.SetBar()\n", []string{"FooUseCase"}},
		{"wired on a different local",
			"foo := usecases.NewFooUseCase(db)\nother.SetBar(barRepo)\n", []string{"FooUseCase"}},
		{"not constructed in New() at all",
			"", nil},
		// Two at once, in reverse alphabetical order in the source. Map iteration
		// is random in Go, so without the sort a failure here would name the same
		// two use cases in a different order each run — and a test that cannot
		// reproduce its own message is one nobody trusts enough to act on.
		{"several unwired come out ordered",
			"zed := usecases.NewZedUseCase(db)\nabel := usecases.NewAbelUseCase(db)\n",
			[]string{"AbelUseCase", "ZedUseCase"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sources := [][]byte{[]byte(decl),
				[]byte("func (uc *ZedUseCase) SetBar(r Bar) {}\nfunc (uc *AbelUseCase) SetBar(r Bar) {}\n")}
			got := unwiredUseCases(sources, []byte(c.wiring), "SetBar")
			if len(got) != len(c.missing) {
				t.Fatalf("got %v, want %v", got, c.missing)
			}
			for i := range got {
				if got[i] != c.missing[i] {
					t.Fatalf("got %v, want %v", got, c.missing)
				}
			}
		})
	}
}
