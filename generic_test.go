package db_test

import (
	"apigo.cc/go/cast"
	"apigo.cc/go/db"
	"apigo.cc/go/file"
	"apigo.cc/go/log"
	_ "modernc.org/sqlite"
	"testing"
)

func TestGenericQuery(t *testing.T) {
	db.ResetAllForTest()
	dbPath := "./test_generic.db"
	file.Remove(dbPath)

	db.SetConfigForTest("test_generic", &db.Config{Type: "sqlite", Host: dbPath})
	dbInst := db.GetDB("test_generic", log.DefaultLogger)
	if dbInst == nil {
		t.Fatal("Failed to get DB")
	}
	defer func() {
		dbInst.Destroy()
		file.Remove(dbPath)
	}()

	r := dbInst.Query("SELECT 1 as num, 'hello' as str")
	res := r.MapOnR1()
	if cast.To[int](res["num"]) != 1 || cast.To[string](res["str"]) != "hello" {
		t.Errorf("cast.To failed, got %v", res)
	}
}
