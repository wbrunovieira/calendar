package persistence

import (
	"database/sql"
)

// Querier is what both *sql.DB and *sql.Tx satisfy. Repositories hold one of these
// rather than a concrete *sql.DB, which is what makes a repository usable inside a
// caller's transaction instead of always opening its own.
//
// Before this, every repository held *sql.DB directly, so a use case spanning several
// writes had NO WAY to be atomic — not by oversight, by construction. A failure at
// instalment 7 of 12 left six committed while the caller was told the whole thing
// failed, and nobody went looking.
type Querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Repositories is the set bound to one unit of work. They share a transaction, so
// writes across them commit or roll back together.
type Repositories struct {
	Transactions *TransactionRepository
	Accounts     *BankAccountRepository
	Invoices     *InvoiceRepository
	Statements   *StatementRepository
	Matches      *MatchRepository
	// Idempotency belongs in the set because the key and the effect must commit
	// together: a key recorded outside the transaction survives a rolled-back effect,
	// and the legitimate retry is then refused as a replay.
	Idempotency *IdempotencyStore
}

// UnitOfWork runs a function inside one database transaction.
type UnitOfWork struct {
	db *sql.DB
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

// Do runs fn inside a transaction, committing if it returns nil and rolling back
// otherwise. A panic rolls back and is re-raised: swallowing it would leave the caller
// believing nothing happened while half the writes had landed.
//
// The repositories handed to fn are bound to that transaction. Using a repository from
// outside it inside fn silently escapes the atomicity, which is why fn receives them
// rather than closing over its own.
func (u *UnitOfWork) Do(fn func(Repositories) error) error {
	tx, err := u.db.Begin()
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := fn(bind(tx)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func bind(q Querier) Repositories {
	return Repositories{
		Transactions: &TransactionRepository{db: q},
		Accounts:     &BankAccountRepository{db: q},
		Invoices:     &InvoiceRepository{db: q},
		Statements:   &StatementRepository{db: q},
		Matches:      &MatchRepository{db: q},
		Idempotency:  &IdempotencyStore{db: q},
	}
}

// beginner is satisfied by *sql.DB but not by *sql.Tx, which is how a repository can
// tell whether it is free to open its own transaction or is already inside one.
type beginner interface {
	Begin() (*sql.Tx, error)
}

// scope gives a repository a transaction to work in, whoever owns it.
//
// A repository that always called Begin could not be composed: nested transactions do
// not exist in Postgres, so an inner Begin on a *sql.Tx is an error, and an inner
// Commit would publish half of the caller's work. Here the OWNER commits; a joined
// scope leaves commit and rollback to whoever opened it.
type scope struct {
	q     Querier
	owned bool
	tx    *sql.Tx
}

func beginScope(q Querier) (*scope, error) {
	if b, ok := q.(beginner); ok {
		tx, err := b.Begin()
		if err != nil {
			return nil, err
		}
		return &scope{q: tx, owned: true, tx: tx}, nil
	}
	// Already inside someone else's transaction: join it.
	return &scope{q: q}, nil
}

// The scope IS a Querier, so a repository body reads the same whether it owns the
// transaction or joined the caller's — no branch at every call site.
func (s *scope) Exec(query string, args ...any) (sql.Result, error) {
	return s.q.Exec(query, args...)
}

func (s *scope) Query(query string, args ...any) (*sql.Rows, error) {
	return s.q.Query(query, args...)
}

func (s *scope) QueryRow(query string, args ...any) *sql.Row {
	return s.q.QueryRow(query, args...)
}

func (s *scope) Commit() error {
	if !s.owned {
		return nil
	}
	return s.tx.Commit()
}

// Rollback is a no-op on a joined scope: undoing the caller's work because an inner
// step failed would be deciding for them. The error travels up instead.
func (s *scope) Rollback() error {
	if s.owned {
		return s.tx.Rollback()
	}
	return nil
}

// DoSimple adapts UnitOfWork to callers that only need the atomicity, not the bound
// repositories — a use case already holding its own repositories, for instance. Those
// repositories must be the ones this UnitOfWork was built from, or their writes land
// outside the transaction and the atomicity is a fiction.
//
// It exists so the application layer can depend on `interface{ Do(func() error) error }`
// without importing this package.
func (u *UnitOfWork) DoSimple(fn func() error) error {
	return u.Do(func(Repositories) error { return fn() })
}
