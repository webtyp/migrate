# migrate

Runs an appkit app's database migrations: the steps of every module, in order, for `webtyp dev`, the self-host supervisor and a pre-deploy.

## Why This Exists

`appkit` is the production declaration of an app and must not carry development or tooling concerns. By extracting the migration logic into this package, one piece is used by the generated `migrate/main.go`, `webtyp dev`, the supervisor, and the database diagram.

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

- `docs/ARCHITECTURE.md`: Details about how migrations are ordered and executed.
