package database

import (
	"regexp"
	"testing"
)

// Migrations run in slice order and nothing resolves dependencies between them.
// A statement referencing a table that a later statement creates fails — but only
// on an empty database. Every developer's database already has the schema, so the
// mistake is invisible locally and surfaces in CI, or in a restore, or on the day
// production is rebuilt.
//
// This happened: reconciliation_matches referenced finance.bank_statement_lines
// while sitting above it in the slice. Local runs were green for days.
//
// The check needs no database, so it runs in the ordinary unit suite.
func TestMigrationOrder_ATableIsCreatedBeforeItIsReferenced(t *testing.T) {
	createRE := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(finance\.[a-z_]+)`)
	referenceRE := regexp.MustCompile(`(?is)REFERENCES\s+(finance\.[a-z_]+)`)

	created := map[string]bool{}
	for i, statement := range migrations() {
		// A table may reference itself inside its own CREATE — a category pointing at
		// its parent, say — so what this statement creates counts as available to it.
		for _, match := range createRE.FindAllStringSubmatch(statement, -1) {
			created[match[1]] = true
		}
		for _, ref := range referenceRE.FindAllStringSubmatch(statement, -1) {
			if !created[ref[1]] {
				t.Errorf("migration %d references %s before anything creates it", i+1, ref[1])
			}
		}
	}
}

// A second table with the same name would mean two definitions racing to be the
// real one, and the loser is silently ignored by CREATE TABLE IF NOT EXISTS.
func TestMigrationOrder_NoTableIsCreatedTwice(t *testing.T) {
	createRE := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(finance\.[a-z_]+)`)

	seen := map[string]int{}
	for i, statement := range migrations() {
		for _, match := range createRE.FindAllStringSubmatch(statement, -1) {
			if first, dup := seen[match[1]]; dup {
				t.Errorf("%s is created twice: migrations %d and %d", match[1], first+1, i+1)
				continue
			}
			seen[match[1]] = i
		}
	}
	if len(seen) == 0 {
		t.Fatal("no CREATE TABLE found: the regex stopped matching and this suite proves nothing")
	}
}
