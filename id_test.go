package db_test

import (
	"os"
	"testing"

	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestAutoRandomID(t *testing.T) {
	dbPath := "id_test.db"
	dbset := "sqlite://" + dbPath
	defer os.Remove(dbPath)

	dbInst := db.GetDB(dbset, nil)
	// Create table with char(12) primary key
	dbInst.Exec("CREATE TABLE test_id (id CHAR(12) PRIMARY KEY, name TEXT)")

	t.Run("AutoFillID", func(t *testing.T) {
		data := map[string]any{"name": "test1"}
		res := dbInst.Insert("test_id", data)
		if res.Error != nil {
			t.Fatalf("Insert failed: %v", res.Error)
		}

		// Verify ID was generated
		qr := dbInst.Query("SELECT id FROM test_id WHERE name='test1'")
		idStr, _ := db.To[string](qr)
		if len(idStr) != 12 {
			t.Errorf("Expected ID length 12, got %d (%s)", len(idStr), idStr)
		}
	})

	t.Run("DoNotOverwriteID", func(t *testing.T) {
		manualID := "manual_id_12"
		data := map[string]any{"id": manualID, "name": "test2"}
		res := dbInst.Insert("test_id", data)
		if res.Error != nil {
			t.Fatalf("Insert failed: %v", res.Error)
		}

		qr := dbInst.Query("SELECT id FROM test_id WHERE name='test2'")
		idStr, _ := db.To[string](qr)
		if idStr != manualID {
			t.Errorf("Expected ID %s, got %s", manualID, idStr)
		}
	})
    
    t.Run("AutoFillEmptyID", func(t *testing.T) {
    	data := map[string]any{"id": "", "name": "test3"}
    	res := dbInst.Insert("test_id", data)
    	if res.Error != nil {
    		t.Fatalf("Insert failed: %v", res.Error)
    	}

    	qr := dbInst.Query("SELECT id FROM test_id WHERE name='test3'")
    	idStr, _ := db.To[string](qr)
    	if len(idStr) != 12 {
    		t.Errorf("Expected ID length 12, got %d (%s)", len(idStr), idStr)
    	}
    })

    t.Run("RedisIDMaker", func(t *testing.T) {
    	// Mock redis config
    	os.Setenv("REDIS_TEST", "redis://:@localhost:6379/1")
    	dbInst.Config.Redis = "test"

    	data := map[string]any{"name": "test_redis"}
    	res := dbInst.Insert("test_id", data)
    	// Even if redis is not running, it should fallback to default id maker or fail gracefully
    	// But here we mainly want to ensure it compiles and runs the logic path
    	if res.Error != nil {
    		t.Logf("Insert with redis config (might fail if no redis): %v", res.Error)
    	} else {
    		qr := dbInst.Query("SELECT id FROM test_id WHERE name='test_redis'")
    		idStr, _ := db.To[string](qr)
    		if len(idStr) != 12 {
    			t.Errorf("Expected ID length 12, got %d (%s)", len(idStr), idStr)
    		}
    	}
    })
    }

