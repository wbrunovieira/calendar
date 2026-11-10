package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A use case that declares SetUnitOfWork is asking for its writes to land
// together or not at all. Nothing checked that anyone actually handed it one.
//
// Mutation proved the gap: deleting the SetUnitOfWork line from New(), and
// making boundTo ignore the bound repositories, both left the whole suite green
// — unit and integration. The atomicity every safety argument on this branch
// rests on was unverified, and a partial conversion was one careless edit away.
//
// It mirrors TestEveryInvoiceRecalculatorAwareUseCaseIsWired: discover the
// capability from the source rather than from a list, so a use case added later
// is covered without anyone remembering to add it here.
func TestEveryUnitOfWorkAwareUseCaseIsWired(t *testing.T) {
	aware := useCasesDeclaring(t, `func \(uc \*(\w+)\) SetUnitOfWork\(`)
	if len(aware) == 0 {
		t.Fatal("found no use case declaring SetUnitOfWork — this guard has stopped guarding anything")
	}

	wiring, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("reading app.go: %v", err)
	}

	constructs := regexp.MustCompile(`(\w+)\s*:?=\s*usecases\.New(\w+)\(`)
	builtFrom := map[string][]string{}
	for _, m := range constructs.FindAllSubmatch(wiring, -1) {
		builtFrom[string(m[2])] = append(builtFrom[string(m[2])], string(m[1]))
	}

	// The ARGUMENT is checked, not just the call: handing a literal nil satisfies
	// "is it called?" while leaving the writes exactly as unprotected.
	wired := regexp.MustCompile(`(\w+)\.SetUnitOfWork\(\s*([&\w{}.]+)`)
	isWired := map[string]bool{}
	for _, m := range wired.FindAllSubmatch(wiring, -1) {
		if string(m[2]) == "nil" {
			continue
		}
		isWired[string(m[1])] = true
	}

	var missing []string
	for typ := range aware {
		locals, built := builtFrom[typ]
		if !built {
			// Constructed inline rather than into a local; nothing to check here,
			// but it cannot be wired either, so say so.
			missing = append(missing, typ+" (never assigned to a local in New)")
			continue
		}
		any := false
		for _, l := range locals {
			if isWired[l] {
				any = true
			}
		}
		if !any {
			missing = append(missing, typ)
		}
	}
	if len(missing) > 0 {
		t.Errorf("these use cases ask for a unit of work and never get one: %s", strings.Join(missing, ", "))
	}
}

// useCasesDeclaring returns the use case types whose source matches the pattern.
func useCasesDeclaring(t *testing.T, pattern string) map[string]bool {
	t.Helper()
	dir := filepath.Join("..", "application", "usecases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the use case package: %v", err)
	}
	re := regexp.MustCompile(pattern)
	found := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		for _, m := range re.FindAllSubmatch(src, -1) {
			found[string(m[1])] = true
		}
	}
	return found
}
