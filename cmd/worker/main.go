package main

import (
	"database/sql"
	"errors"
	"flag"
	"gitlab.com/digineat/go-broker-test/internal/db"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "data.db", "path to sqlite database")
	poll := flag.Duration("poll", 100*time.Millisecond, "poll interval")
	flag.Parse()

	sqlDB, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err = db.InitDB(sqlDB); err != nil {
		log.Fatal(err)
	}

	log.Printf("worker polling every %v", *poll)
	for range time.Tick(*poll) {
		if err = db.ProcessNext(sqlDB); err != nil && !errors.Is(err, db.ErrNoTrade) {
			log.Println("process error:", err)
		}
	}
}
