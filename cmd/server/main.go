package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"gitlab.com/digineat/go-broker-test/internal/db"
	"gitlab.com/digineat/go-broker-test/internal/model"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "data.db", "path to sqlite database")
	listen := flag.String("listen", ":8080", "HTTP listen address")
	flag.Parse()

	sqlDB, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.InitDB(sqlDB); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler(sqlDB))
	mux.HandleFunc("/trades", postTradesHandler(sqlDB))
	mux.HandleFunc("/stats/", getStatsHandler(sqlDB))

	addr := fmt.Sprintf("%s", *listen)
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func healthHandler(sqlDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := sqlDB.Ping(); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func postTradesHandler(sqlDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var ti model.TradeInput
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&ti); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if !ti.Validate() {
			http.Error(w, "invalid data", http.StatusBadRequest)
			return
		}
		if err := db.EnqueueTrade(sqlDB, ti); err != nil {
			http.Error(w, "enqueue failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getStatsHandler(sqlDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		acc := strings.TrimPrefix(r.URL.Path, "/stats/")
		if acc == "" {
			http.Error(w, "account required", http.StatusBadRequest)
			return
		}
		stats, err := db.GetStats(sqlDB, acc)
		if err != nil {
			http.Error(w, "error fetching stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}
