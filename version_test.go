package db_test

import (
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
		ver, _ = db.ToValue[int64](qr)
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
		ver, _ = db.ToValue[int64](qr)
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
