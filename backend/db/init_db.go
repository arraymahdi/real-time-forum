package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() {
	var err error
	Instance, err = sql.Open("sqlite3", "./forum.db")
	if err != nil {
		log.Fatal(err)
	}
}
