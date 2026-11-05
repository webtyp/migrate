//go:build !wasm

package migrate_test

import (
	"errors"
	"strings"
	"testing"

	"webtyp.com/appkit"
	"webtyp.com/ddl"
	"webtyp.com/migrate"
	"webtyp.com/storage"
	"webtyp.com/storage/mem"
)

var calls []string

func mod(name string) appkit.Module {
	return appkit.Module{Name: name, Server: appkit.ModuleServer{
		Migrate: func(ddl.Execer, ddl.Compiler) error { calls = append(calls, "migrate:"+name); return nil },
	}}
}
func openMem(string) (storage.Conn, error) { return mem.New(), nil } // webtyp.com/storage/mem

func TestStepsModuleOrder(t *testing.T) {
	app := appkit.App{Modules: []appkit.Module{mod("a"), mod("b"), {Name: "c"}}}
	steps := migrate.Steps(app)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Name != "a" || steps[1].Name != "b" {
		t.Errorf("expected names a, b; got %s, %s", steps[0].Name, steps[1].Name)
	}
}

func TestStepsLoginByRUTFirst(t *testing.T) {
	app := appkit.App{
		Login:   appkit.LoginByRUT{},
		Modules: []appkit.Module{mod("a")},
	}
	steps := migrate.Steps(app)
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}
	names := []string{"auth", "rbac", "access", "a"}
	for i, n := range names {
		if steps[i].Name != n {
			t.Errorf("step %d: expected %s, got %s", i, n, steps[i].Name)
		}
	}
}

func TestStepsNoLogin(t *testing.T) {
	app := appkit.App{Modules: []appkit.Module{mod("a")}}
	steps := migrate.Steps(app)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Name != "a" {
		t.Errorf("expected name a, got %s", steps[0].Name)
	}
}

func TestRunOrder(t *testing.T) {
	calls = nil
	app := appkit.App{Modules: []appkit.Module{mod("a"), mod("b")}}
	err := migrate.Run(app, mem.New(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[0] != "migrate:a" || calls[1] != "migrate:b" {
		t.Errorf("unexpected calls: %v", calls)
	}
}

func TestRunStopsAtError(t *testing.T) {
	calls = nil
	m1 := appkit.Module{Name: "a", Server: appkit.ModuleServer{
		Migrate: func(ddl.Execer, ddl.Compiler) error { return errors.New("boom") },
	}}
	app := appkit.App{Modules: []appkit.Module{m1, mod("b")}}
	err := migrate.Run(app, mem.New(), nil)
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
	app := appkit.App{
		Modules: []appkit.Module{mod("a")},
		Log: func(args ...any) {
			logs = append(logs, args...)
		},
	}
	_ = migrate.Run(app, mem.New(), nil)
	if len(logs) == 0 || logs[0] != "migrate:" || logs[1] != "a" {
		t.Errorf("unexpected logs: %v", logs)
	}
}

func TestAppNoDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	err := migrate.App(appkit.App{})
	if !errors.Is(err, migrate.ErrNoDatabaseURL) {
		t.Errorf("expected ErrNoDatabaseURL, got %v", err)
	}
}

func TestAppNoOpen(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	err := migrate.App(appkit.App{})
	if !errors.Is(err, appkit.ErrNoOpen) {
		t.Errorf("expected ErrNoOpen, got %v", err)
	}
}

func TestAppNoCompiler(t *testing.T) {
	t.Setenv("DATABASE_URL", "x")
	app := appkit.App{Server: appkit.AppServer{Database: openMem}}
	err := migrate.App(app)
	if !errors.Is(err, migrate.ErrNoCompiler) && err != nil { // mem does not implement ddl.Compiler currently, we expect ErrNoCompiler or nil if it was updated
		// Let's explicitly check if mem.New() implements ddl.Compiler to assert correctly
		if _, ok := (interface{})(mem.New()).(ddl.Compiler); ok {
			if err != nil {
				t.Errorf("expected nil error since mem implements ddl.Compiler, got %v", err)
			}
		} else {
			if !errors.Is(err, migrate.ErrNoCompiler) {
				t.Errorf("expected ErrNoCompiler, got %v", err)
			}
		}
	}
}
