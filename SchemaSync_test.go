package db_test

import (
	"os"
	"testing"

	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestSchemaSync(t *testing.T) {
	dbPath := "test_schema.db"
	dbInst := db.GetDB("sqlite://"+dbPath, nil)
	defer os.Remove(dbPath)
	defer dbInst.Exec("DROP TABLE IF EXISTS test_table")
	defer dbInst.Exec("DROP TABLE IF EXISTS test_table_deleted")

	schema := `== Default ==
test_table SD // Test table with shadow delete
  id AI // ID
  name v50 U
  autoVersion ubi // Version
  status ti // Status
`
	err := dbInst.Sync(schema)
	if err != nil {
		t.Fatal("Sync error:", err)
	}

	dbInst.Insert("test_table", map[string]any{"name": "test", "status": 1})

	dbInst.Delete("test_table", "id=?", 1)

	res := dbInst.Query("SELECT COUNT(*) FROM test_table_deleted")
	if res.IntOnR1C1() != 1 {
		t.Fatal("Shadow delete failed")
	}
}

func TestAutoDetectShadow(t *testing.T) {
	dbPath := "auto_detect.db"
	dbInst := db.GetDB("sqlite://"+dbPath, nil)
	defer os.Remove(dbPath)
	defer dbInst.Exec("DROP TABLE IF EXISTS test_auto")
	defer dbInst.Exec("DROP TABLE IF EXISTS test_auto_deleted")

	// Manually create tables, DO NOT call Sync
	dbInst.Exec("CREATE TABLE test_auto (id INTEGER PRIMARY KEY)")
	dbInst.Exec("CREATE TABLE test_auto_deleted (id INTEGER PRIMARY KEY)")
	dbInst.Insert("test_auto", map[string]any{"id": 1})

	// This should trigger auto-detection and perform a shadow delete
	dbInst.Delete("test_auto", "id=?", 1)

	res := dbInst.Query("SELECT COUNT(*) FROM test_auto_deleted")
	if res.IntOnR1C1() != 1 {
		t.Fatal("Auto-detect shadow delete failed")
	}
}

