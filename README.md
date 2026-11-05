# migrate
<img src="docs/img/badges.svg">

Runs an appkit app's database migrations: the steps of every module, in order, for `webtyp dev`, the self-host supervisor and a pre-deploy.

## Why This Exists

- **Without it:** [`appkit`](https://github.com/webtyp/appkit), the production declaration of an app, also builds and runs its migrations.
- **With it:** one piece migrates an app — the generated `migrate/main.go`, `webtyp dev`, the self-host supervisor and the database diagram all use it. Each step runs [`ddl.Sync`](https://github.com/webtyp/ddl): it adds tables and columns and never deletes rows.

## Before You Start

This package is server-only. It opens a database using `App.Server.Database` and the `DATABASE_URL` environment variable.

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
	if err := migrate.App(config.App); err != nil {
		log.Fatal(err)
	}
}
```

The database diagram tool reads the list of migrations using `migrate.Steps`.

## Documentation Map

| Document | What it explains |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Order of the steps (login, then modules) and stop at the first error |
| [migrate.go](migrate.go) | `App`, `Run`, `Steps` and the errors |
