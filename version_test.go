package db_test

import (
	"os"
	"testing"

	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestVersionControl(t *testing.T) {
	dbInst := db.GetDB("sqlite://:memory:", nil)

	// Create table with autoVersion
	dbInst.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, autoVersion BIGINT UNSIGNED)")

	t.Run("InsertAutoVersion", func(t *testing.T) {
		data := map[string]any{"id": 1, "name": "Alice"}
		res := dbInst.Insert("users", data)
		if res.Error != nil {
			t.Fatalf("Insert failed: %v", res.Error)
		}

		// Verify version was injected
		var ver int64
		qr := dbInst.Query("SELECT autoVersion FROM users WHERE id = 1")
		ver, _ = db.To[int64](qr)
		if ver != 1 {
			t.Errorf("Expected version 1, got %d", ver)
		}
	})

	t.Run("UpdateOptimisticLock", func(t *testing.T) {
		// First update
		data := map[string]any{"name": "Alice Updated", "autoVersion": int64(1)}
		res := dbInst.Update("users", data, "id = 1")
		if res.Error != nil {
			t.Fatalf("Update failed: %v", res.Error)
		}
		if res.Changes() != 1 {
			t.Errorf("Expected 1 change, got %d", res.Changes())
		}

		// Verify version incremented
		var ver int64
		qr := dbInst.Query("SELECT autoVersion FROM users WHERE id = 1")
		ver, _ = db.To[int64](qr)
		if ver != 2 {
			t.Errorf("Expected version 2, got %d", ver)
		}

		// Try update with old version (should fail to update any rows)
		dataConflict := map[string]any{"name": "Conflict", "autoVersion": int64(1)}
		resConflict := dbInst.Update("users", dataConflict, "id = 1")
		if resConflict.Changes() != 0 {
			t.Errorf("Expected 0 changes due to optimistic lock, got %d", resConflict.Changes())
		}
	})
}

func TestVersionInitialization(t *testing.T) {
	dbPath := "init_test.db"
	dbset := "sqlite://" + dbPath
	defer os.Remove(dbPath)

	dbInst := db.GetDB(dbset, nil)
	dbInst.Exec("CREATE TABLE test_init (id INTEGER PRIMARY KEY, autoVersion BIGINT UNSIGNED)")

	// Manually insert with a high version
	dbInst.Exec("INSERT INTO test_init (id, autoVersion) VALUES (1, 100)")

	// First insert via DB helper should pick up 101
	data := map[string]any{"id": 2}
	res := dbInst.Insert("test_init", data)
	if res.Error != nil {
		t.Fatalf("Insert failed: %v", res.Error)
	}

	ver, _ := db.To[int64](dbInst.Query("SELECT autoVersion FROM test_init WHERE id=2"))
	if ver != 101 {
		t.Errorf("Expected version 101, got %d", ver)
	}

	// Update should make it 102
	dbInst.Update("test_init", map[string]any{"autoVersion": 101}, "id=2")
	ver, _ = db.To[int64](dbInst.Query("SELECT autoVersion FROM test_init WHERE id=2"))
	if ver != 102 {
		t.Errorf("Expected version 102, got %d", ver)
	}
}
