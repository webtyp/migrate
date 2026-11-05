//go:build !wasm

// Package migrate runs database migrations: the steps passed to it, in order.
// It is the whole body of an app's generated migrate/main.go.
package migrate

import (
	"errors"
	"fmt"

	"webtyp.com/ddl"
	"webtyp.com/env"
	"webtyp.com/storage"
)

const (
	// EnvDatabaseURL is the environment variable that holds the database connection string.
	EnvDatabaseURL = "DATABASE_URL"
)

var (
	// ErrNoDatabaseURL: there is no database address to migrate.
	ErrNoDatabaseURL = errors.New("migrate: DATABASE_URL is not set")
	// ErrNoOpen: DATABASE_URL is set but open function is nil.
	ErrNoOpen = errors.New("migrate: DATABASE_URL is set but open is nil")
	// ErrNoCompiler: the connection cannot turn a table change into SQL.
	ErrNoCompiler = errors.New("migrate: the database connection does not implement ddl.Compiler")
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

// App opens DATABASE_URL with open and runs every migration step in order.
//
// Example (migrate/main.go): if err := migrate.App(config.App.Server.Database, log.Println, config.Migrations...); err != nil { log.Fatal(err) }
func App(open func(dsn string) (storage.Conn, error), log func(...any), steps ...ddl.Step) error {
	dsn := env.Get(EnvDatabaseURL)
	if dsn == "" {
		return ErrNoDatabaseURL
	}
	if open == nil {
		return ErrNoOpen
	}
	conn, err := open(dsn)
	if err != nil {
		return err
	}
	compiler, ok := conn.(ddl.Compiler)
	if !ok {
		return ErrNoCompiler
	}
	return Run(conn, compiler, log, steps...)
}
