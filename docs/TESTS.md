# Tests

| Test | Rule |
|---|---|
| [TestRunOrder](../tests/migrate_test.go) | Run runs the steps in slice order |
| [TestRunStopsAtError](../tests/migrate_test.go) | Run stops at the first failing step and names it |
| [TestRunLogs](../tests/migrate_test.go) | Run logs each step's name before running it |
| [TestAppNoDatabaseURL](../tests/migrate_test.go) | App refuses to run without DATABASE_URL |
| [TestAppNoOpen](../tests/migrate_test.go) | App refuses a nil opener |
| [TestAppOpenError](../tests/migrate_test.go) | App returns the opener's error |
| [TestAppNoCompiler](../tests/migrate_test.go) | App refuses a connection that cannot compile DDL |
| [TestAppSuccess](../tests/migrate_test.go) | App captures the steps, then runs them on the database in order |
| [TestLeftovers_AppReportsColumnWithData](../tests/leftovers_test.go) | App returns a column with data the model no longer has |
| [TestLeftovers_AppCopiesMovedToWithoutDropping](../tests/leftovers_test.go) | App copies a leftover into its MovedTo column and never drops it |
| [TestLeftovers_AppCannotProbe](../tests/leftovers_test.go) | App fails loudly when the connection cannot list columns |
| [TestLeftovers_DropRemovesColumn](../tests/leftovers_test.go) | Drop removes the column, data included |
| [TestLeftovers_DropCopiesBeforeDropping](../tests/leftovers_test.go) | Drop copies into MovedTo before dropping |
| [TestLeftovers_DropNeedsTableAndColumn](../tests/leftovers_test.go) | Drop rejects a target that is not table.column |
| [TestLeftovers_PlanLineIsJSON](../tests/leftovers_test.go) | PlanLine prints one leftover as one JSON object |
