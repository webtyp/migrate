---
PLAN: "feat: migrate an appkit app — App (from DATABASE_URL), Run (an open connection), Steps (the ordered list)"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 2036931737131336487
PR: https://github.com/webtyp/migrate/pull/1
REVIEWED_COMMIT: 33c0c13ed00109050478227a631fefce4b13ddd4
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan — one piece owns migrating an app's database

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/appkit@v0.2.8 webtyp.com/ddl@v0.0.39 webtyp.com/env@v0.1.3 webtyp.com/storage@v0.1.18 \
       webtyp.com/access@v0.0.41 webtyp.com/auth@v0.0.102 webtyp.com/rbac@v0.0.29
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`, `go mod tidy` after `go get`.

## Why

`appkit` is the production declaration of an app and must not carry development or tooling concerns (rule in
project-layout). Today it also builds and runs the app's migrations (`appkit/migrate.go`: `Steps`, `RunMigrations`,
`Migrate`). This repo takes that job, so the generated `migrate/main.go` of every app becomes `migrate.App(config.App)`,
and `webtyp dev`, the self-host supervisor and the database diagram all use one piece. A later plan in `appkit` deletes
its copy.

## Ecosystem search

- `appkit/migrate.go` (appkit v0.2.8): the code to **move** (body below). Reused types: `appkit.App`,
  `appkit.LoginByRUT`, `appkit.Module.Server.Migrate`, `appkit.AppServer.Database`, `appkit.EnvDatabaseURL`,
  `appkit.ErrNoOpen` (both stay in appkit: its server uses them too).
- `ddl.Step{Name string; Run func(conn ddl.Execer, compiler ddl.Compiler) error}` (`ddl/capture.go:11`),
  `ddl.Execer`, `ddl.Compiler` → reuse.
- Login tables: `authority.Migrate` (`webtyp.com/auth/authority`), `rbac.Migrate` (`webtyp.com/rbac`),
  `migrate.Migrate` of `webtyp.com/access/migrate` (import it as `accessmigrate`).

## Design gate

1. Prior art (unverified): Rails `bin/rails db:migrate`, Django `manage.py migrate`, Ecto `mix ecto.migrate` — one
   command, separate from serving, run before the new version starts.
2. Names: `migrate.App(app)` reads «migrate the app»; `migrate.Run(app, conn, compiler)` for a caller with an open
   connection (an app's tests, an importer); `migrate.Steps(app)` the list (the diagram reads it).
3. Ledger: concepts moved, not added; ways to migrate an app stay 1 once appkit deletes its copy.
4. Belongs here: one concern — migrating an app. Server-only.
5. Deletes: the gonew scaffold `migrate.go` (`type Migrate`, `New`). `appkit`'s copy is deleted by its own plan.

## APIs used

| Symbol | Signature | Tag |
|---|---|---|
| `appkit.App` | fields used: `Login appkit.Login` (interface), `Modules []appkit.Module`, `Server appkit.AppServer`, `Log func(...any)` | appkit v0.2.8 |
| `appkit.Module` | `Name string`, `Server appkit.ModuleServer` | appkit v0.2.8 |
| `appkit.ModuleServer.Migrate` | `func(conn ddl.Execer, compiler ddl.Compiler) error` (nil = no tables) | appkit v0.2.8 |
| `appkit.AppServer.Database` | `func(dsn string) (storage.Conn, error)` | appkit v0.2.8 |
| `appkit.EnvDatabaseURL` | `const = "DATABASE_URL"` | appkit v0.2.8 |
| `appkit.ErrNoOpen` | `error` | appkit v0.2.8 |
| `appkit.LoginByRUT` | struct type | appkit v0.2.8 |
| `env.Get` | `func Get(key string) string` | env v0.1.3 |
| `authority.Migrate`, `rbac.Migrate`, `accessmigrate.Migrate` | `func(conn ddl.Execer, compiler ddl.Compiler) error` | as pinned |

If a signature in the published tag differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stage 1 — `migrate.go` (replace the scaffold), `//go:build !wasm`

```go
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
```

If `storage.Conn` does not satisfy `ddl.Execer` in that tag, stop and name it in the PR.

## Stage 2 — tests (`tests/migrate_test.go`, package `migrate_test`, `//go:build !wasm`)

Helper (copy):
```go
var calls []string
func mod(name string) appkit.Module {
	return appkit.Module{Name: name, Server: appkit.ModuleServer{
		Migrate: func(ddl.Execer, ddl.Compiler) error { calls = append(calls, "migrate:"+name); return nil },
	}}
}
func openMem(string) (storage.Conn, error) { return mem.New(), nil } // webtyp.com/storage/mem
```

| Test | Setup → call | Expected |
|---|---|---|
| `TestStepsModuleOrder` | `Modules: {mod("a"), mod("b"), {Name:"c"}}` → `Steps` | 2 steps, names `a`, `b` |
| `TestStepsLoginByRUTFirst` | `Login: appkit.LoginByRUT{}`, `Modules: {mod("a")}` → `Steps` | names `auth`, `rbac`, `access`, `a` |
| `TestStepsNoLogin` | `Modules: {mod("a")}` → `Steps` | names `a` |
| `TestRunOrder` | `calls = nil`; `{mod("a"), mod("b")}` → `Run(app, mem.New(), nil)` | `nil`; `calls == ["migrate:a","migrate:b"]` |
| `TestRunStopsAtError` | module "a" whose Migrate returns `errors.New("boom")`, then `mod("b")` → `Run` | error contains `"migrate a: boom"`; `calls` has no `"migrate:b"` |
| `TestRunLogs` | `Log` records its args; `{mod("a")}` → `Run` | first record is `["migrate:", "a"]` |
| `TestAppNoDatabaseURL` | `t.Setenv("DATABASE_URL", "")` → `App(appkit.App{})` | `errors.Is(err, migrate.ErrNoDatabaseURL)` |
| `TestAppNoOpen` | `t.Setenv("DATABASE_URL", "x")` → `App(appkit.App{})` | `errors.Is(err, appkit.ErrNoOpen)` |
| `TestAppNoCompiler` | `t.Setenv("DATABASE_URL", "x")`; `Server: appkit.AppServer{Database: openMem}` → `App` | `errors.Is(err, migrate.ErrNoCompiler)` (if `mem` implements `ddl.Compiler`, assert `nil` instead and say so in the PR) |

## Stage 3 — docs (English, webtyp templates)

`README.md` (≤150 lines, 4 sections): Why This Exists (without: appkit carries migration; with: one piece used by
`migrate/main.go`, `webtyp dev`, the supervisor and the diagram), Before You Start (server only; `DATABASE_URL`;
`App.Server.Database`), Step-by-Step (the generated `migrate/main.go`:
`func main() { if err := migrate.App(config.App); err != nil { log.Fatal(err) } }`; reading `Steps`), Documentation
Map. `docs/ARCHITECTURE.md` (4 sections): order of steps (login, then modules), stop at first error, `ddl.Sync` inside
each step keeps rows.

## Delivery check

1. `gotest` passes. A PR that does not compile, or whose tests could not run, is a failed delivery.
2. `grep -rn "type Migrate\|func New" --include=*.go .` → empty.

Commit after every stage. Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own
work. Stuck 30 minutes: open the PR anyway listing what fails.

## Stages

| Stage | Files | Done when |
|---|---|---|
| 1 | `migrate.go`, `go.mod`, `go.sum` | compiles |
| 2 | `tests/migrate_test.go` | tests pass |
| 3 | `README.md`, `docs/ARCHITECTURE.md` | done |
