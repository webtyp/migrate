# migrate
<img src="docs/img/badges.svg">

Runs database migrations: the steps passed to it, in order, for `webtyp dev`, the self-host supervisor and a pre-deploy.

## Why This Exists

- **Without it:** [`appkit`](https://github.com/webtyp/appkit), the production declaration of an app, also builds and runs its migrations.
- **With it:** one piece migrates an app — the generated `migrate/main.go`, `webtyp dev`, the self-host supervisor and the database diagram all use it. Each step runs [`ddl.Sync`](https://github.com/webtyp/ddl): it adds tables and columns and never deletes rows.

## Before You Start

This package is server-only. It opens a database using an opener function and the `DATABASE_URL` environment variable.

## Step-by-Step

The code below is the generated `migrate/main.go` of every app:

```go
package main

import (
	"log"

	"webtyp.com/migrate"
	"yourdomain/config"
)

func main() {
	if err := migrate.App(config.App.Server.Database, log.Println, config.Migrations...); err != nil {
		log.Fatal(err)
	}
}
```

The database diagram tool reads the list of migrations using `config.Migrations`.

## Documentation Map

| Document | What it explains |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Order of the steps and stop at the first error |
| [migrate.go](migrate.go) | `App`, `Run` and the errors |
