//go:build !wasm

package migrate_test

import (
	"errors"
	"path/filepath"
	"testing"

	"webtyp.com/ddl"
	"webtyp.com/migrate"
	"webtyp.com/model"
	"webtyp.com/sqlite"
	"webtyp.com/storage"
)

// holiday is a model whose table the test creates by hand with one more column, "label".
type holiday struct{}

func (*holiday) ModelName() string { return "holiday" }
func (*holiday) Schema() []model.Field {
	return []model.Field{{Name: "id", Type: model.Text()}, {Name: "name", Type: model.Text()}}
}
func (*holiday) Pointers() []any                { return nil }
func (h *holiday) IsNil() bool                  { return h == nil }
func (*holiday) EncodeFields(model.FieldWriter) {}
func (*holiday) DecodeFields(model.FieldReader) {}

var holidayStep = ddl.Step{Name: "calendar", Run: func(conn ddl.Execer, c ddl.Compiler) error {
	return ddl.New(conn, c).Sync(&holiday{})
}}

// holidayDB is a sqlite file with table holiday(id, name, label): row h1 has only a label,
// row h2 has a name and a label. DATABASE_URL points to it.
func holidayDB(t *testing.T) storage.Conn {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "app.db")
	t.Setenv(migrate.EnvDatabaseURL, dsn)
	conn, err := sqlite.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	for _, q := range []string{
		"CREATE TABLE holiday (id TEXT PRIMARY KEY, name TEXT, label TEXT)",
		"INSERT INTO holiday (id, name, label) VALUES ('h1', NULL, 'Navidad')",
		"INSERT INTO holiday (id, name, label) VALUES ('h2', 'Año nuevo', 'Year')",
	} {
		if err := conn.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return conn
}

func names(t *testing.T, conn storage.Conn) []string {
	t.Helper()
	rows, err := conn.Query("SELECT name FROM holiday ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func columns(t *testing.T, conn storage.Conn) []string {
	t.Helper()
	cols, err := conn.(ddl.TableIntrospector).TableColumns("holiday")
	if err != nil {
		t.Fatal(err)
	}
	return cols
}

func TestLeftovers_AppReportsColumnWithData(t *testing.T) {
	holidayDB(t)
	left, err := migrate.App(sqlite.Open, nil, ddl.Schema{Steps: []ddl.Step{holidayStep}})
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0].Table != "holiday" || left[0].Column != "label" || left[0].Rows != 2 {
		t.Fatalf("want one leftover holiday.label with 2 rows, got %+v", left)
	}
}

func TestLeftovers_AppCopiesMovedToWithoutDropping(t *testing.T) {
	conn := holidayDB(t)
	removed := []ddl.Removed{{Table: "holiday", Column: "label", MovedTo: "name"}}
	if _, err := migrate.App(sqlite.Open, nil, ddl.Schema{Steps: []ddl.Step{holidayStep}, Removed: removed}); err != nil {
		t.Fatal(err)
	}
	if got := names(t, conn); len(got) != 2 || got[0] != "Navidad" || got[1] != "Año nuevo" {
		t.Fatalf("want [Navidad Año nuevo], got %v", got)
	}
	if got := columns(t, conn); len(got) != 3 {
		t.Fatalf("label must stay: want 3 columns, got %v", got)
	}
}

func TestLeftovers_AppCannotProbe(t *testing.T) {
	t.Setenv(migrate.EnvDatabaseURL, "x")
	open := func(string) (storage.Conn, error) { return &dummyCompilerConn{Conn: memConn()}, nil }
	if _, err := migrate.App(open, nil, ddl.Schema{}); err == nil {
		t.Fatal("want an error: the connection cannot list columns")
	}
}

func TestLeftovers_DropRemovesColumn(t *testing.T) {
	conn := holidayDB(t)
	if err := migrate.Drop(sqlite.Open, ddl.Schema{}, "holiday.label"); err != nil {
		t.Fatal(err)
	}
	if got := columns(t, conn); len(got) != 2 {
		t.Fatalf("want [id name], got %v", got)
	}
	if got := names(t, conn); got[0] != "" {
		t.Fatalf("no MovedTo: h1 keeps an empty name, got %q", got[0])
	}
}

func TestLeftovers_DropCopiesBeforeDropping(t *testing.T) {
	conn := holidayDB(t)
	removed := []ddl.Removed{{Table: "holiday", Column: "label", MovedTo: "name"}}
	if err := migrate.Drop(sqlite.Open, ddl.Schema{Removed: removed}, "holiday.label"); err != nil {
		t.Fatal(err)
	}
	if got := names(t, conn); got[0] != "Navidad" || got[1] != "Año nuevo" {
		t.Fatalf("want [Navidad Año nuevo], got %v", got)
	}
	if got := columns(t, conn); len(got) != 2 {
		t.Fatalf("want [id name], got %v", got)
	}
}

func TestLeftovers_DropNeedsTableAndColumn(t *testing.T) {
	for _, target := range []string{"", "holiday", "holiday.", ".label"} {
		if err := migrate.Drop(sqlite.Open, ddl.Schema{}, target); !errors.Is(err, migrate.ErrDropTarget) {
			t.Errorf("%q: want ErrDropTarget, got %v", target, err)
		}
	}
}

func TestLeftovers_PlanLineIsJSON(t *testing.T) {
	l := ddl.Leftover{Table: "holiday", Column: "label", Rows: 2, Samples: []string{"Navidad", "Year"},
		Removed: ddl.Removed{Table: "holiday", Column: "label", MovedTo: "name"}}
	want := `{"table":"holiday","column":"label","rows":2,"more":false,"samples":["Navidad","Year"],"moved_to":"name","note":""}`
	if got := migrate.PlanLine(l); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	empty := migrate.PlanLine(ddl.Leftover{Table: "t", Column: "c", Rows: 1})
	if empty != `{"table":"t","column":"c","rows":1,"more":false,"samples":[],"moved_to":"","note":""}` {
		t.Fatalf("got %s", empty)
	}
}
