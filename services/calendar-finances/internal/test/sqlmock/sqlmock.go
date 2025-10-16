package sqlmock

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sync"
	"time"
)

type expectationKind int

const (
	kindBegin expectationKind = iota
	kindExec
	kindQuery
	kindCommit
)

type expectation struct {
	kind   expectationKind
	query  *regexp.Regexp
	args   []interface{}
	result driver.Result
	rows   *Rows
}

type Mock struct {
	expectations []*expectation
	index        int
}

type mockDriver struct {
	mock *Mock
}

type mockConn struct {
	mock *Mock
}

type mockTx struct {
	mock *Mock
}

var (
	driverCounter int
	driverMu      sync.Mutex
)

// New creates a new sql.DB backed by the test driver and a Mock handle to configure expectations.
func New() (*sql.DB, *Mock, error) {
	driverMu.Lock()
	defer driverMu.Unlock()

	name := fmt.Sprintf("mocksql-%d", driverCounter)
	driverCounter++

	mock := &Mock{}
	sql.Register(name, &mockDriver{mock: mock})
	db, err := sql.Open(name, "")
	if err != nil {
		return nil, nil, err
	}

	return db, mock, nil
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{mock: d.mock}, nil
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("mock driver does not support Prepare")
}

func (c *mockConn) Close() error { return nil }

func (c *mockConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *mockConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if _, err := c.mock.fulfill(kindBegin, "", nil); err != nil {
		return nil, err
	}
	return &mockTx{mock: c.mock}, nil
}

func (c *mockConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return c.ExecContext(context.Background(), query, named)
}

func (c *mockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.mock.fulfill(kindExec, query, namedToValues(args))
}

func (c *mockConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return c.QueryContext(context.Background(), query, named)
}

func (c *mockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	res, rows, err := c.mock.fulfillWithRows(kindQuery, query, namedToValues(args))
	if err != nil {
		return nil, err
	}
	_ = res // exec result not used
	return rows, nil
}

func (c *mockConn) CheckNamedValue(*driver.NamedValue) error { return nil }

func (tx *mockTx) Commit() error {
	_, err := tx.mock.fulfill(kindCommit, "", nil)
	return err
}

func (tx *mockTx) Rollback() error {
	return nil
}

func (m *Mock) fulfill(kind expectationKind, query string, args []driver.Value) (driver.Result, error) {
	res, _, err := m.fulfillWithRows(kind, query, args)
	return res, err
}

func (m *Mock) fulfillWithRows(kind expectationKind, query string, args []driver.Value) (driver.Result, driver.Rows, error) {
	if m.index >= len(m.expectations) {
		return nil, nil, fmt.Errorf("unexpected call: %s", query)
	}
	exp := m.expectations[m.index]
	if exp.kind != kind {
		return nil, nil, fmt.Errorf("unexpected expectation order: got %v, want %v", kindToString(kind), kindToString(exp.kind))
	}
	if exp.query != nil && !exp.query.MatchString(query) {
		return nil, nil, fmt.Errorf("query %q does not match expectation %q", query, exp.query.String())
	}
	if len(exp.args) > 0 {
		if len(exp.args) != len(args) {
			return nil, nil, fmt.Errorf("argument count mismatch: got %d, want %d", len(args), len(exp.args))
		}
		for i, expected := range exp.args {
			if !matchArg(expected, args[i]) {
				return nil, nil, fmt.Errorf("argument %d mismatch: got %v, want %v", i, args[i], expected)
			}
		}
	}
	m.index++

	if exp.rows != nil {
		return exp.result, exp.rows.iterator(), nil
	}
	return exp.result, nil, nil
}

func namedToValues(values []driver.NamedValue) []driver.Value {
	res := make([]driver.Value, len(values))
	for i, v := range values {
		res[i] = v.Value
	}
	return res
}

// ExpectBegin registers an expected BEGIN call.
func (m *Mock) ExpectBegin() {
	m.expectations = append(m.expectations, &expectation{kind: kindBegin})
}

// ExpectCommit registers an expected COMMIT call.
func (m *Mock) ExpectCommit() {
	m.expectations = append(m.expectations, &expectation{kind: kindCommit})
}

// ExpectExec registers an expected Exec statement.
func (m *Mock) ExpectExec(query string) *ExpectedExec {
	return &ExpectedExec{exp: m.appendExpectation(kindExec, query)}
}

// ExpectQuery registers an expected Query statement.
func (m *Mock) ExpectQuery(query string) *ExpectedQuery {
	return &ExpectedQuery{exp: m.appendExpectation(kindQuery, query)}
}

func (m *Mock) appendExpectation(kind expectationKind, query string) *expectation {
	var re *regexp.Regexp
	if query != "" {
		re = regexp.MustCompile(query)
	}
	exp := &expectation{kind: kind, query: re}
	m.expectations = append(m.expectations, exp)
	return exp
}

// ExpectationsWereMet asserts that all expectations have been fulfilled.
func (m *Mock) ExpectationsWereMet() error {
	if m.index != len(m.expectations) {
		return fmt.Errorf("there are %d unfulfilled expectations", len(m.expectations)-m.index)
	}
	return nil
}

// ExpectedExec allows configuring Exec behaviour.
type ExpectedExec struct {
	exp *expectation
}

// WithArgs configures the expected arguments for Exec.
func (e *ExpectedExec) WithArgs(args ...interface{}) *ExpectedExec {
	e.exp.args = args
	return e
}

// WillReturnResult configures the Exec to return a specific result.
func (e *ExpectedExec) WillReturnResult(result driver.Result) {
	e.exp.result = result
}

// ExpectedQuery allows configuring Query behaviour.
type ExpectedQuery struct {
	exp *expectation
}

// WithArgs configures expected arguments for the query.
func (q *ExpectedQuery) WithArgs(args ...interface{}) *ExpectedQuery {
	q.exp.args = args
	return q
}

// WillReturnRows configures the rows returned by the query.
func (q *ExpectedQuery) WillReturnRows(rows *Rows) {
	q.exp.rows = rows.clone()
}

// Rows represents a deterministic set of query results for expectations.
type Rows struct {
	columns []string
	data    [][]driver.Value
}

// NewRows creates a new Rows definition with the provided columns.
func NewRows(columns []string) *Rows {
	return &Rows{columns: append([]string(nil), columns...)}
}

// AddRow appends a row to the set.
func (r *Rows) AddRow(values ...interface{}) *Rows {
	row := make([]driver.Value, len(values))
	for i, v := range values {
		row[i] = v
	}
	r.data = append(r.data, row)
	return r
}

func (r *Rows) clone() *Rows {
	copyRows := make([][]driver.Value, len(r.data))
	for i, row := range r.data {
		copyRows[i] = append([]driver.Value(nil), row...)
	}
	return &Rows{columns: append([]string(nil), r.columns...), data: copyRows}
}

func (r *Rows) iterator() driver.Rows {
	return &rowsIterator{columns: r.columns, data: r.data}
}

type rowsIterator struct {
	columns []string
	data    [][]driver.Value
	index   int
}

func (r *rowsIterator) Columns() []string { return append([]string(nil), r.columns...) }

func (r *rowsIterator) Close() error { return nil }

func (r *rowsIterator) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}
	row := r.data[r.index]
	for i := range dest {
		dest[i] = row[i]
	}
	r.index++
	return nil
}

// Result implements driver.Result for deterministic responses.
type Result struct {
	lastInsertID int64
	rowsAffected int64
}

// NewResult creates a driver.Result with the provided values.
func NewResult(lastInsertID, rowsAffected int64) driver.Result {
	return &Result{lastInsertID: lastInsertID, rowsAffected: rowsAffected}
}

func (r *Result) LastInsertId() (int64, error) { return r.lastInsertID, nil }

func (r *Result) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type anyArgument struct{}

// AnyArg matches any argument value.
func AnyArg() interface{} { return anyArgument{} }

func matchArg(expected interface{}, actual driver.Value) bool {
	switch exp := expected.(type) {
	case anyArgument:
		return true
	case time.Time:
		if at, ok := actual.(time.Time); ok {
			return exp.Equal(at)
		}
		return false
	default:
		return reflect.DeepEqual(exp, actual)
	}
}

func kindToString(k expectationKind) string {
	switch k {
	case kindBegin:
		return "BEGIN"
	case kindExec:
		return "EXEC"
	case kindQuery:
		return "QUERY"
	case kindCommit:
		return "COMMIT"
	default:
		return "UNKNOWN"
	}
}

var errNotSupported = errors.New("operation not supported")

func (c *mockConn) PrepareContext(context.Context, string) (driver.Stmt, error) {
	return nil, errNotSupported
}

func (c *mockConn) Ping(context.Context) error { return nil }

func (c *mockConn) ResetSession(context.Context) error { return nil }

var _ driver.ExecerContext = (*mockConn)(nil)
var _ driver.QueryerContext = (*mockConn)(nil)
var _ driver.ConnBeginTx = (*mockConn)(nil)
var _ driver.NamedValueChecker = (*mockConn)(nil)
