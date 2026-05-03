package db_test

import (
	"testing"
	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestSmartDelete(t *testing.T) {
	dbInst := db.GetDB("sqlite://:memory:", nil)
	
	// Create table and shadow table
	dbInst.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, item TEXT)")
	dbInst.Exec("CREATE TABLE orders_deleted (id INTEGER PRIMARY KEY, item TEXT)")

	t.Run("ShadowDelete", func(t *testing.T) {
		dbInst.Exec("INSERT INTO orders (id, item) VALUES (1, 'Phone')")
		
		res := dbInst.Delete("orders", "id = 1")
		if res.Error != nil {
			t.Fatalf("Delete failed: %v", res.Error)
		}
		if res.Changes() != 1 {
			t.Errorf("Expected 1 change, got %d", res.Changes())
		}

		// Verify it's gone from main table
		qr := dbInst.Query("SELECT COUNT(*) FROM orders WHERE id = 1")
		count, _ := db.ToValue[int](qr)
		if count != 0 {
			t.Errorf("Expected 0 records in main table, got %d", count)
		}

		// Verify it's in shadow table
		qr2 := dbInst.Query("SELECT COUNT(*) FROM orders_deleted WHERE id = 1")
		countDeleted, _ := db.ToValue[int](qr2)
		if countDeleted != 1 {
			t.Errorf("Expected 1 record in shadow table, got %d", countDeleted)
		}
	})

	t.Run("PhysicalDelete", func(t *testing.T) {
		dbInst.Exec("CREATE TABLE logs (id INTEGER PRIMARY KEY, msg TEXT)")
		dbInst.Exec("INSERT INTO logs (id, msg) VALUES (1, 'Login')")

		dbInst.Delete("logs", "id = 1")
		
		qr := dbInst.Query("SELECT COUNT(*) FROM logs WHERE id = 1")
		count, _ := db.ToValue[int](qr)
		if count != 0 {
			t.Errorf("Expected 0 records in logs, got %d", count)
		}
	})
}
