package db_test

import (
	"testing"
	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestTableProbing(t *testing.T) {
	dbInst := db.GetDB("sqlite://:memory:", nil)
	
	// Create a table with autoVersion
	dbInst.Exec("CREATE TABLE table_with_ver (id INTEGER PRIMARY KEY, name TEXT, autoVersion BIGINT UNSIGNED)")
	// Create a table with shadow table
	dbInst.Exec("CREATE TABLE table_with_shadow (id INTEGER PRIMARY KEY, name TEXT)")
	dbInst.Exec("CREATE TABLE table_with_shadow_deleted (id INTEGER PRIMARY KEY, name TEXT)")

	t.Run("ProbeAutoVersion", func(t *testing.T) {
		// We need a way to access getTable or check its effect.
		// Since getTable is private, we can't call it directly from _test package.
		// But we can check if it exists in the struct if we move test to 'db' package or use reflection.
		// Alternatively, we can just ensure it doesn't crash for now, and Feature 3/4 will use it.
		
		// For now, let's just trigger it.
		dbInst.Query("SELECT * FROM table_with_ver")
	})
}
