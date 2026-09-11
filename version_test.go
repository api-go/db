package db_test

import (
	"testing"
	"time"

	"apigo.cc/go/db"
	"apigo.cc/go/file"
	"apigo.cc/go/log"
	_ "modernc.org/sqlite"
)

func TestVersionControl(t *testing.T) {
	db.ResetAllForTest()
	dbPath := "./test_version.db"
	file.Remove(dbPath)
	db.SetConfigForTest("test_version", &db.Config{Type: "sqlite", Host: dbPath})
	dbInst := db.GetDB("test_version", log.DefaultLogger)
	if dbInst == nil {
		t.Fatal("db is nil")
	}
	defer func() {
		dbInst.Destroy()
		file.Remove(dbPath)
	}()

	dbInst.Exec("CREATE TABLE versioned_docs (id INTEGER PRIMARY KEY, content TEXT, autoVersion BIGINT)")

	// Initial insert
	res := dbInst.Insert("versioned_docs", map[string]string{"content": "v1"})
	if res.Error != nil {
		t.Fatalf("Insert failed: %v", res.Error)
	}
	if res.Id() != 1 {
		t.Fatalf("Expected ID 1, got %d", res.Id())
	}

	// Check initial version
	v1 := dbInst.Query("SELECT autoVersion FROM versioned_docs WHERE id=1").IntOnR1C1()
	if v1 <= 0 {
		t.Errorf("Expected initial version > 0, got %d", v1)
	}

	// Update should increment version
	time.Sleep(1 * time.Millisecond) // Ensure NextVersion has a different timestamp if needed by underlying implementation
	updateRes := dbInst.Update("versioned_docs", map[string]string{"content": "v2"}, "id=?", 1)
	if updateRes.Error != nil {
		t.Fatalf("Update failed: %v", updateRes.Error)
	}

	v2 := dbInst.Query("SELECT autoVersion FROM versioned_docs WHERE id=1").IntOnR1C1()
	if v2 <= v1 {
		t.Errorf("Expected version to increment, got v2=%d, v1=%d", v2, v1)
	}
}
