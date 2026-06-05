package db

import (
	"fmt"
	"os"
	"testing"
)

func TestCheckTable_DuplicateUnique(t *testing.T) {
	dbPath := "test_unique_race.db"
	_ = os.Remove(dbPath)
	defer os.Remove(dbPath)

	dbInst := GetDB("sqlite://"+dbPath, nil)
	
	schema := `
== System ==
_Table SD
  id c10 PK
  name v64 U
`
    // First sync
	err := dbInst.Sync(schema)
	if err != nil {
		t.Fatalf("First sync failed: %v", err)
	}

	// Insert duplicates to ensure recreation fails
	dbInst.Exec("INSERT INTO _Table (id, name) VALUES ('1', 'dup')")
	dbInst.Exec("INSERT INTO _Table (id, name) VALUES ('2', 'dup')") // Will fail if unique constraint works, but wait, the unique index is already there, so we can't insert duplicates! 
	// Wait, we CAN'T insert duplicates because the first sync created the index.
	// But in the user's log, there ARE duplicates? Or maybe there are NO duplicates, but the engine just complains when it recreates the index on existing data?
	// Ah, if there are NO duplicates, CREATE UNIQUE INDEX will SUCCEED.
	// So why did the user get UNIQUE constraint failed? 
	// BECAUSE THERE WERE DUPLICATES!
	// Why would there be duplicates in _Table?
	// Let's just do the second sync and see if it tries to execute CREATE UNIQUE INDEX.
	
	// Print statements will be inside Schema.go for a moment.
	
    // Second sync (simulate restart)
    err = dbInst.Sync(schema)
	if err != nil {
		t.Fatalf("Second sync failed: %v", err)
	}
}
