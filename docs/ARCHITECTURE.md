# Architecture

## Order of Steps

`App` lists the tables the steps create (`ddl.Capture`, which runs each step once against a recorder that touches no database), opens `DATABASE_URL` with the opener, runs every step in slice order, copies each leftover column that `removed` moves to another column, and returns the columns the database keeps although no model has them (they hold data).

See [MIGRATE_FLOW](diagrams/MIGRATE_FLOW.md).

## Stop at First Error

If a step fails to run, `App` stops at the first error.

## Leftover Columns

`App` never drops a column. It copies `MovedTo` on every run, so production repeats the copy without dropping; only rows where the target is empty are filled. It returns only columns with data.

## Dropping a Column

Dropping a column happens only via `Drop` and only by a person's decision. It copies first when `MovedTo` is set.