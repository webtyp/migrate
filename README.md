# migrate
<img src="docs/img/badges.svg">

Runs database migrations: the steps passed to it, in order, for `webtyp dev`, the self-host supervisor and a pre-deploy.

## Why This Exists

Without: a removed field leaves its column and its data in the database and nobody sees it.
With: `migrate` runs the steps, copies what [`ddl.Removed`](https://github.com/webtyp/ddl/blob/main/leftovers.go) moves, and returns the leftover columns; dropping one is a separate, explicit `-drop`.

## Before You Start

This package is server-only. It opens a database using an opener function (`postgres.Open`, `sqlite.Open`) and the `DATABASE_URL` environment variable.

## Step-by-Step Guide

The code below is the generated `migrate/main.go` of every app:

```go
package main

import (
	"flag"
	"fmt"
	"log"

	"webtyp.com/migrate"

	"yourdomain/config"
)

func main() {
	plan := flag.Bool(migrate.FlagPlan, false, "print one JSON line per leftover column")
	drop := flag.String(migrate.FlagDrop, "", "drop one column, data included: table.column")
	flag.Parse()
	if *drop != "" {
		if err := migrate.Drop(config.App.Server.Database, config.Schema, *drop); err != nil {
			log.Fatal(err)
		}
		return
	}
	left, err := migrate.App(config.App.Server.Database, log.Println, config.Schema)
	if err != nil {
		log.Fatal(err)
	}
	if *plan {
		for _, l := range left {
			fmt.Println(migrate.PlanLine(l))
		}
	}
}
```

Then one `-plan` output line:

```json
{"table":"holiday","column":"label","rows":2,"more":false,"samples":["Navidad","Year"],"moved_to":"name","note":""}
```

See [migrate.go](migrate.go) for the errors.

## Documentation Map

| Document | What it explains |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Architecture details |
| [docs/diagrams/MIGRATE_FLOW.md](docs/diagrams/MIGRATE_FLOW.md) | Order of operations diagram |
| [docs/TESTS.md](docs/TESTS.md) | Test map |
| [migrate.go](migrate.go) | `App`, `Drop`, `Run` and the errors |