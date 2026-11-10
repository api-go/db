package db_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/api-go/cast"
	"github.com/api-go/db"
)

func TestStandaloneIndexes(t *testing.T) {
	store := db.GetDB("sqlite://"+filepath.Join(t.TempDir(), "indexes.db"), nil)
	defer store.Destroy()
	base := "records SD\n  id i AI\n  a v32\n  b v32\n  status ti I\n"
	indexes := "  a,b U\n  b,a I\n  a,-b I\n  -b I\n  a I\n"
	for i := 0; i < 2; i++ {
		if err := store.Sync(base + indexes); err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, index := range store.Query("PRAGMA index_list(records)").MapResults() {
			columns := []string{}
			for _, field := range store.Query("PRAGMA index_xinfo(" + store.Quote(cast.String(index["name"])) + ")").MapResults() {
				if cast.Int(field["key"]) == 0 {
					continue
				}
				name := cast.String(field["name"])
				if cast.Int(field["desc"]) != 0 {
					name = "-" + name
				}
				columns = append(columns, name)
			}
			found[cast.String(index["unique"])+":"+strings.Join(columns, ",")] = true
		}
		for _, expected := range []string{"1:a,b", "0:b,a", "0:a,-b", "0:-b", "0:status", "0:a"} {
			if !found[expected] {
				t.Fatalf("missing %s: %v", expected, found)
			}
		}
		if len(found) != 6 {
			t.Fatalf("duplicate or extra indexes: %v", found)
		}
	}
	if result := store.Exec("INSERT INTO records(a,b) VALUES('x','y')"); result.Error != nil {
		t.Fatal(result.Error)
	}
	if result := store.Exec("INSERT INTO records(a,b) VALUES('x','y')"); result.Error == nil {
		t.Fatal("composite unique constraint missing")
	}
	plan := store.Query("EXPLAIN QUERY PLAN SELECT b FROM records WHERE a=? ORDER BY b DESC LIMIT 10", "x").MapResults()
	for _, row := range plan {
		if strings.Contains(cast.String(row["detail"]), "TEMP B-TREE") {
			t.Fatalf("descending query sorts instead of using index: %v", plan)
		}
	}
	if indexes := store.Query("PRAGMA index_list(records_deleted)").MapResults(); len(indexes) != 0 {
		t.Fatalf("shadow table must not copy live constraints: %v", indexes)
	}
	// Declarations removed from the schema remove their physical indexes.
	if err := store.Sync(base + "  b,-a I\n"); err != nil {
		t.Fatal(err)
	}
	if indexes := store.Query("PRAGMA index_list(records)").MapResults(); len(indexes) != 2 {
		t.Fatalf("stale indexes remain: %v", indexes)
	}
}

func TestStandaloneIndexValidationBeforeMutation(t *testing.T) {
	store := db.GetDB("sqlite://"+filepath.Join(t.TempDir(), "invalid-index.db"), nil)
	defer store.Destroy()
	base := "records\n  id i AI\n  a v32\n  b v32\n"
	if err := store.Sync(base); err != nil {
		t.Fatal(err)
	}
	for _, declaration := range []string{"a,missing I", "a,a U", "a, I", "a,b BOGUS", "-a TI", "a DESC,b I", "missing TI", "missing", "missing bogus", "missing v0", "missing v32 BAD", "missing t I U", "missing dt ct ctu", "missing i nn n", "missing AI"} {
		// A rejected index must not add a requested new column first.
		if err := store.Sync(base + "  pending v32\n  " + declaration + "\n"); err == nil {
			t.Fatalf("accepted %q", declaration)
		}
		for _, field := range store.Query("PRAGMA table_info(records)").MapResults() {
			if field["name"] == "pending" {
				t.Fatalf("invalid %q changed the schema", declaration)
			}
		}
	}
}
