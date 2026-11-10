package db_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/api-go/db"
	_ "modernc.org/sqlite"
)

func TestQueryDynamicStorageClasses(t *testing.T) {
	db.ResetAllForTest()
	db.SetConfigForTest("dynamic_rows", &db.Config{Type: "sqlite", Host: filepath.Join(t.TempDir(), "values.db")})
	database := db.GetDB("dynamic_rows", nil)
	defer database.Destroy()
	// A column may contain integer, fractional, text and NULL values. Its first
	// driver's ScanType must not coerce the later rows into that first type.
	const query = "SELECT 2 AS value UNION ALL SELECT 0.125 UNION ALL SELECT 'text' UNION ALL SELECT NULL"
	want := []any{int64(2), float64(0.125), "text", nil}
	t.Run("maps", func(t *testing.T) {
		result := database.Query(query)
		rows := result.MapResults()
		if result.Error != nil {
			t.Fatal(result.Error)
		}
		got := make([]any, len(rows))
		for i, row := range rows {
			got[i] = row["value"]
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})
	t.Run("slices", func(t *testing.T) {
		result := database.Query(query)
		rows := result.SliceResults()
		if result.Error != nil {
			t.Fatal(result.Error)
		}
		got := make([]any, len(rows))
		for i, row := range rows {
			got[i] = row[0]
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("structs", func(t *testing.T) {
		rows, err := db.ToSlice[struct{ Value any }](database.Query(query))
		if err != nil {
			t.Fatal(err)
		}
		got := make([]any, len(rows))
		for i, row := range rows {
			got[i] = row.Value
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("scalar_interfaces", func(t *testing.T) {
		rows, err := db.ToSlice[any](database.Query(query))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rows, want) {
			t.Fatalf("got %#v", rows)
		}
	})
	t.Run("typed_float", func(t *testing.T) {
		rows, err := db.ToSlice[struct{ Value float64 }](database.Query("SELECT 2 AS value UNION ALL SELECT 0.125"))
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || rows[0].Value != 2 || rows[1].Value != 0.125 {
			t.Fatalf("got %#v", rows)
		}
	})
}

func TestQueryDateTimePreservesTime(t *testing.T) {
	db.ResetAllForTest()
	db.SetConfigForTest("datetime_rows", &db.Config{Type: "sqlite", Host: filepath.Join(t.TempDir(), "dates.db")})
	database := db.GetDB("datetime_rows", nil)
	defer database.Destroy()
	if r := database.Exec("CREATE TABLE samples (occurred DATETIME, day DATE)"); r.Error != nil {
		t.Fatal(r.Error)
	}
	if r := database.Exec("INSERT INTO samples VALUES (?,?)", "2026-10-07T14:49:12.809Z", "2026-10-07"); r.Error != nil {
		t.Fatal(r.Error)
	}
	result := database.Query("SELECT occurred, day FROM samples")
	rows := result.MapResults()
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if len(rows) != 1 || rows[0]["occurred"] != "2026-10-07 14:49:12.809" || rows[0]["day"] != "2026-10-07" {
		t.Fatalf("datetime was truncated: %#v", rows)
	}
}
