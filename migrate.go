//go:build !wasm

// Package migrate runs database migrations: the steps passed to it, in order.
// It is the whole body of an app's generated migrate/main.go.
package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"webtyp.com/ddl"
	"webtyp.com/env"
	"webtyp.com/storage"
)

const (
	// EnvDatabaseURL is the environment variable that holds the database connection string.
	EnvDatabaseURL = "DATABASE_URL"
	// FlagPlan is the flag of migrate/main.go that prints one PlanLine per leftover column.
	FlagPlan = "plan"
	// FlagDrop is the flag of migrate/main.go that drops one column, data included: -drop table.column.
	FlagDrop = "drop"

	columnSeparator = "."
)

var (
	// ErrNoDatabaseURL: there is no database address to migrate.
	ErrNoDatabaseURL = errors.New("migrate: DATABASE_URL is not set")
	// ErrNoOpen: DATABASE_URL is set but open function is nil.
	ErrNoOpen = errors.New("migrate: DATABASE_URL is set but open is nil")
	// ErrNoCompiler: the connection cannot turn a table change into SQL.
	ErrNoCompiler = errors.New("migrate: the database connection does not implement ddl.Compiler")
	// ErrDropTarget: -drop needs table.column.
	ErrDropTarget = errors.New("migrate: -drop needs table.column")
)

// Run runs the given steps on an open connection, stopping at the first error.
//
// Example: err := migrate.Run(conn, compiler, log.Println, steps...)
func Run(conn ddl.Execer, compiler ddl.Compiler, log func(...any), steps ...ddl.Step) error {
	for _, step := range steps {
		if log != nil {
			log("migrate:", step.Name)
		}
		if err := step.Run(conn, compiler); err != nil {
			return fmt.Errorf("migrate %s: %w", step.Name, err)
		}
	}
	return nil
}

// App lists the tables the steps create (ddl.Capture, which runs each step once against a
// recorder that touches no database), opens DATABASE_URL with open, runs every step in order, copies each
// leftover column that schema.Removed moves to another column, and returns the columns the
// database keeps although no model has them (they hold data). It never drops a column.
//
// Example (migrate/main.go): left, err := migrate.App(config.App.Server.Database, log.Println, config.Schema)
func App(open func(dsn string) (storage.Conn, error), log func(...any), schema ddl.Schema) ([]ddl.Leftover, error) {
	tables, err := ddl.Capture(schema.Steps)
	if err != nil {
		return nil, err
	}
	conn, compiler, err := connect(open)
	if err != nil {
		return nil, err
	}
	if err := Run(conn, compiler, log, schema.Steps...); err != nil {
		return nil, err
	}
	db := ddl.New(conn, compiler)
	left, err := db.Leftovers(tables, schema.Removed)
	if err != nil {
		return nil, err
	}
	for _, l := range left {
		if l.Removed.MovedTo == "" {
			continue
		}
		if err := db.CopyColumn(l.Removed); err != nil {
			return nil, err
		}
	}
	return left, nil
}

// Drop opens DATABASE_URL with open and drops one column, data included: target is
// "table.column". When schema.Removed moves that column to another one, it copies it first.
// Only for a decision a person made (webtyp dev's «yes»).
//
// Example (migrate/main.go): err := migrate.Drop(config.App.Server.Database, config.Schema, "holiday.label")
func Drop(open func(dsn string) (storage.Conn, error), schema ddl.Schema, target string) error {
	table, column, ok := strings.Cut(target, columnSeparator)
	if !ok || table == "" || column == "" {
		return ErrDropTarget
	}
	conn, compiler, err := connect(open)
	if err != nil {
		return err
	}
	db := ddl.New(conn, compiler)
	for _, r := range schema.Removed {
		if r.Table == table && r.Column == column && r.MovedTo != "" {
			if err := db.CopyColumn(r); err != nil {
				return err
			}
		}
	}
	return db.DropColumn(table, column)
}

// planLine is the JSON shape of one PlanLine.
type planLine struct {
	Table   string   `json:"table"`
	Column  string   `json:"column"`
	Rows    int      `json:"rows"`
	More    bool     `json:"more"`
	Samples []string `json:"samples"`
	MovedTo string   `json:"moved_to"`
	Note    string   `json:"note"`
}

// PlanLine is one line of `migrate -plan`: a leftover column as one JSON object, so
// webtyp dev and an agent read it without parsing prose.
//
//	{"table":"holiday","column":"label","rows":2,"more":false,"samples":["a","b"],"moved_to":"name","note":""}
func PlanLine(l ddl.Leftover) string {
	samples := l.Samples
	if samples == nil {
		samples = []string{}
	}
	b, _ := json.Marshal(planLine{
		Table: l.Table, Column: l.Column, Rows: l.Rows, More: l.More, Samples: samples,
		MovedTo: l.Removed.MovedTo, Note: l.Removed.Note,
	})
	return string(b)
}

func connect(open func(dsn string) (storage.Conn, error)) (storage.Conn, ddl.Compiler, error) {
	dsn := env.Get(EnvDatabaseURL)
	if dsn == "" {
		return nil, nil, ErrNoDatabaseURL
	}
	if open == nil {
		return nil, nil, ErrNoOpen
	}
	conn, err := open(dsn)
	if err != nil {
		return nil, nil, err
	}
	compiler, ok := conn.(ddl.Compiler)
	if !ok {
		return nil, nil, ErrNoCompiler
	}
	return conn, compiler, nil
}
