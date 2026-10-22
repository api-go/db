package db_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"apigo.cc/go/db"
	_ "modernc.org/sqlite"
)

func TestQueryJSONByteStorageRetainsObjectsAndOrdinaryBinary(t *testing.T) {
	db.SetConfigForTest("json_byte_rows", &db.Config{Type: "sqlite", Host: filepath.Join(t.TempDir(), "values.db")})
	database := db.GetDB("json_byte_rows", nil)
	defer database.Destroy()
	// BLOB storage under JSON/JSONB declarations reproduces pgx's byte-valued
	// JSON columns without requiring a network database in the normal gate.
	if r := database.Exec("CREATE TABLE samples (settings JSONB, tags JSON, raw BLOB)"); r.Error != nil {
		t.Fatal(r.Error)
	}
	object := []byte(` {"semantics":["Password"],"enabled":true}`)
	array := []byte(`["one","two"]`)
	if r := database.Exec("INSERT INTO samples VALUES (?,?,?)", object, array, object); r.Error != nil {
		t.Fatal(r.Error)
	}
	wantObject := map[string]any{"semantics": []any{"Password"}, "enabled": true}
	wantTags := []any{"one", "two"}
	result := database.Query("SELECT settings,tags,raw FROM samples")
	rows := result.MapResults()
	if result.Error != nil || len(rows) != 1 {
		t.Fatal(rows, result.Error)
	}
	if !reflect.DeepEqual(rows[0]["settings"], wantObject) || !reflect.DeepEqual(rows[0]["tags"], wantTags) {
		t.Fatalf("declared JSON returned driver bytes: %#v", rows[0])
	}
	if !reflect.DeepEqual(rows[0]["raw"], object) {
		t.Fatalf("ordinary binary was interpreted as JSON: %#v", rows[0]["raw"])
	}
	binaryMaps, err := db.ToSlice[map[string][]byte](database.Query("SELECT settings,tags,raw FROM samples"))
	if err != nil || len(binaryMaps) != 1 || !reflect.DeepEqual(binaryMaps[0]["settings"], object) || !reflect.DeepEqual(binaryMaps[0]["tags"], array) {
		t.Fatalf("explicit binary map: %#v, %v", binaryMaps, err)
	}
	binaryRows, err := db.ToSlice[[][]byte](database.Query("SELECT settings,tags,raw FROM samples"))
	if err != nil || len(binaryRows) != 1 || !reflect.DeepEqual(binaryRows[0], [][]byte{object, array, object}) {
		t.Fatalf("explicit binary slice: %#v, %v", binaryRows, err)
	}
	values, err := db.ToSlice[struct{ Settings, Tags, Raw any }](database.Query("SELECT settings,tags,raw FROM samples"))
	if err != nil || len(values) != 1 || !reflect.DeepEqual(values[0].Settings, wantObject) || !reflect.DeepEqual(values[0].Tags, wantTags) || !reflect.DeepEqual(values[0].Raw, object) {
		t.Fatalf("interface fields: %#v, %v", values, err)
	}
	if r := database.Exec("INSERT INTO samples VALUES (?,?,?)", []byte(`{invalid`), nil, []byte(`[]`)); r.Error != nil {
		t.Fatal(r.Error)
	}
	invalid := database.Query("SELECT settings,tags FROM samples WHERE tags IS NULL").MapOnR1()
	if !reflect.DeepEqual(invalid["settings"], []byte(`{invalid`)) || invalid["tags"] != nil {
		t.Fatalf("invalid JSON or SQL NULL was changed: %#v", invalid)
	}
}

func TestQueryFixedCharacterPaddingDoesNotCreatePhantomReferences(t *testing.T) {
	db.SetConfigForTest("fixed_character_rows", &db.Config{Type: "sqlite", Host: filepath.Join(t.TempDir(), "values.db")})
	database := db.GetDB("fixed_character_rows", nil)
	defer database.Destroy()
	if r := database.Exec("CREATE TABLE samples (empty BPCHAR, reference CHAR(10), note VARCHAR(20), body TEXT)"); r.Error != nil {
		t.Fatal(r.Error)
	}
	if r := database.Exec("INSERT INTO samples VALUES (?,?,?,?)", "          ", " key      ", "note  ", "body  "); r.Error != nil {
		t.Fatal(r.Error)
	}
	result := database.Query("SELECT empty,reference,note,body FROM samples")
	row := result.MapOnR1()
	if result.Error != nil || row["empty"] != "" || row["reference"] != " key" || row["note"] != "note  " || row["body"] != "body  " {
		t.Fatalf("fixed character padding escaped or meaningful text changed: %#v %v", row, result.Error)
	}
	typed, err := db.ToSlice[struct{ Empty, Reference, Note, Body string }](database.Query("SELECT empty,reference,note,body FROM samples"))
	if err != nil || len(typed) != 1 || typed[0].Empty != "" || typed[0].Reference != " key" || typed[0].Note != "note  " || typed[0].Body != "body  " {
		t.Fatalf("typed fixed character normalization: %#v %v", typed, err)
	}
}
