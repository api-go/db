package main

import (
	"fmt"
	"apigo.cc/go/db"
)

func main() {
	dbPath := "debug.db"
	dbInst := db.GetDB("sqlite://"+dbPath, nil)
	schema := `
== System ==
_Table SD
  id c10 PK
  name v64 U
`
	fmt.Println("First sync...")
	dbInst.Sync(schema)

	fmt.Println("\nSecond sync...")
	dbInst.Sync(schema)
}
