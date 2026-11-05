//go:build !wasm

package migrate_test

import (
	"errors"
	"strings"
	"testing"

	"webtyp.com/ddl"
	"webtyp.com/migrate"
	"webtyp.com/model"
	"webtyp.com/storage"
	"webtyp.com/storage/mem"
)

var calls []string

type dummyCompilerConn struct {
	storage.Conn
}

func (d *dummyCompilerConn) CompileDDL(s ddl.Stmt, m model.Model) (string, []any, error) {
	return "", nil, nil
}

func step(name string) ddl.Step {
	return ddl.Step{
		Name: name,
		Run: func(ddl.Execer, ddl.Compiler) error {
			calls = append(calls, "migrate:"+name)
			return nil
		},
	}
}

func TestRunOrder(t *testing.T) {
	calls = nil
	err := migrate.Run(mem.New(), nil, nil, step("a"), step("b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[0] != "migrate:a" || calls[1] != "migrate:b" {
		t.Errorf("unexpected calls: %v", calls)
	}
}

func TestRunStopsAtError(t *testing.T) {
	calls = nil
	failingStep := ddl.Step{
		Name: "a",
		Run:  func(ddl.Execer, ddl.Compiler) error { return errors.New("boom") },
	}
	err := migrate.Run(mem.New(), nil, nil, failingStep, step("b"))
	if err == nil || !strings.Contains(err.Error(), "migrate a: boom") {
		t.Errorf("expected 'migrate a: boom', got %v", err)
	}
	for _, call := range calls {
		if call == "migrate:b" {
			t.Errorf("unexpected call to migrate:b")
		}
	}
}

func TestRunLogs(t *testing.T) {
	var logs []any
	logger := func(args ...any) {
		logs = append(logs, args...)
	}
	_ = migrate.Run(mem.New(), nil, logger, step("a"))
	if len(logs) == 0 || logs[0] != "migrate:" || logs[1] != "a" {
		t.Errorf("unexpected logs: %v", logs)
	}
}

func TestAppNoDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	err := migrate.App(nil, nil)
	if !errors.Is(err, migrate.ErrNoDatabaseURL) {
		t.Errorf("expected ErrNoDatabaseURL, got %v", err)
	}
}

func TestAppNoOpen(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	err := migrate.App(nil, nil)
	if !errors.Is(err, migrate.ErrNoOpen) {
		t.Errorf("expected ErrNoOpen, got %v", err)
	}
}

func TestAppOpenError(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	openErr := errors.New("fail open")
	open := func(string) (storage.Conn, error) {
		return nil, openErr
	}
	err := migrate.App(open, nil)
	if !errors.Is(err, openErr) {
		t.Errorf("expected openErr, got %v", err)
	}
}

func TestAppNoCompiler(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	openMem := func(string) (storage.Conn, error) { return mem.New(), nil }
	err := migrate.App(openMem, nil)
	if !errors.Is(err, migrate.ErrNoCompiler) {
		t.Errorf("expected ErrNoCompiler, got %v", err)
	}
}

func TestAppSuccess(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	calls = nil
	openDummy := func(string) (storage.Conn, error) {
		return &dummyCompilerConn{Conn: mem.New()}, nil
	}
	err := migrate.App(openDummy, nil, step("a"), step("b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[0] != "migrate:a" || calls[1] != "migrate:b" {
		t.Errorf("unexpected calls: %v", calls)
	}
}
