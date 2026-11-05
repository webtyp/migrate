# Architecture

## Order of Steps

Migrations are run in a specific order: the steps passed to `migrate.App` or `migrate.Run`, in slice order (typically `config.Migrations`).

## Stop at First Error

The migration runner iterates through the steps on an open connection. It stops and returns an error immediately at the first step that fails.

## Sync Keeps Rows

Inside each step, `ddl.Sync` is used, which keeps existing rows intact while updating the table schema.
