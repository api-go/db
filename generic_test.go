package db_test

import (
	"testing"
	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestGenericQuery(t *testing.T) {
	dbInst := db.GetDB("sqlite://:memory:", nil)
	if dbInst == nil {
		t.Fatal("Failed to get DB")
	}

	dbInst.Exec("CREATE TABLE test_generic (id INTEGER PRIMARY KEY, name TEXT)")
	dbInst.Exec("INSERT INTO test_generic (name) VALUES (?)", "Alice")
	dbInst.Exec("INSERT INTO test_generic (name) VALUES (?)", "Bob")

	t.Run("ToSlice", func(t *testing.T) {
		type Item struct {
			Id   int
			Name string
		}
		res := dbInst.Query("SELECT id, name FROM test_generic ORDER BY id")
		items, err := db.ToSlice[Item](res)
		if err != nil {
			t.Fatalf("ToSlice failed: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}
		if items[0].Name != "Alice" || items[1].Name != "Bob" {
			t.Errorf("Incorrect data: %+v", items)
		}
	})

	t.Run("ToValue", func(t *testing.T) {
		res := dbInst.Query("SELECT name FROM test_generic WHERE id = ?", 1)
		name, err := db.ToValue[string](res)
		if err != nil {
			t.Fatalf("ToValue failed: %v", err)
		}
		if name != "Alice" {
			t.Errorf("Expected Alice, got %s", name)
		}
	})
}
