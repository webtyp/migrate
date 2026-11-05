//go:build !wasm

// Package migrate migrates the database of an appkit.App: the tables of its login and of each module, in
// App.Modules order. It is the whole body of an app's generated migrate/main.go.
package migrate

import (
	"errors"
	"fmt"

	accessmigrate "webtyp.com/access/migrate"
	"webtyp.com/appkit"
	"webtyp.com/auth/authority"
	"webtyp.com/ddl"
	"webtyp.com/env"
	"webtyp.com/rbac"
)

var (
	// ErrNoDatabaseURL: there is no database address to migrate.
	ErrNoDatabaseURL = errors.New("migrate: DATABASE_URL is not set")
	// ErrNoCompiler: the connection cannot turn a table change into SQL.
	ErrNoCompiler = errors.New("migrate: the database connection does not implement ddl.Compiler")
)

// Steps are the migrations of the app: its login's tables first, then each module's, in App.Modules order.
// A module without Migrate has none.
//
// Example: steps := migrate.Steps(config.App)
func Steps(app appkit.App) []ddl.Step {
	var out []ddl.Step
	switch app.Login.(type) {
	case appkit.LoginByRUT:
		out = append(out,
			ddl.Step{Name: "auth", Run: authority.Migrate},
			ddl.Step{Name: "rbac", Run: rbac.Migrate},
			ddl.Step{Name: "access", Run: accessmigrate.Migrate},
		)
	}
	for _, m := range app.Modules {
		if m.Server.Migrate != nil {
			out = append(out, ddl.Step{Name: m.Name, Run: m.Server.Migrate})
		}
	}
	return out
}

// Run runs Steps(app) on an open connection, stopping at the first error.
//
// Example: err := migrate.Run(config.App, conn, compiler)
func Run(app appkit.App, conn ddl.Execer, compiler ddl.Compiler) error {
	for _, step := range Steps(app) {
		if app.Log != nil {
			app.Log("migrate:", step.Name)
		}
		if err := step.Run(conn, compiler); err != nil {
			return fmt.Errorf("migrate %s: %w", step.Name, err)
		}
	}
	return nil
}

// App opens DATABASE_URL with App.Server.Database and runs every migration of the app.
//
// Example (migrate/main.go): if err := migrate.App(config.App); err != nil { log.Fatal(err) }
func App(app appkit.App) error {
	dsn := env.Get(appkit.EnvDatabaseURL)
	if dsn == "" {
		return ErrNoDatabaseURL
	}
	if app.Server.Database == nil {
		return appkit.ErrNoOpen
	}
	conn, err := app.Server.Database(dsn)
	if err != nil {
		return err
	}
	compiler, ok := conn.(ddl.Compiler)
	if !ok {
		return ErrNoCompiler
	}
	return Run(app, conn, compiler)
}
