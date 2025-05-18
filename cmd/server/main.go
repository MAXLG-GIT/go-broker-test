package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/go-playground/validator/v10"
	_ "github.com/mattn/go-sqlite3"
	dbmanager "gitlab.com/digineat/go-broker-test/internal/db"
	"gitlab.com/digineat/go-broker-test/internal/model"
	"log"
	"net/http"
	"regexp"
)

func main() {

	//log.SetOutput(os.Stdout)
	//
	//// настраиваем префикс и флаги: время + файл:строка
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	//log.SetPrefix("[MYAPP] ")

	// Command line flags
	dbPath := flag.String("db", "data.db", "path to SQLite database")
	listenAddr := flag.String("listen", "8080", "HTTP server listen address")
	flag.Parse()

	// Initialize database connection
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer func(db *sql.DB) {
		err = db.Close()
		if err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}(db)

	// Test database connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	dbManager := dbmanager.Manager{}
	err = dbManager.InitDbManager(db)
	if err != nil {
		log.Fatalf("Can not init DB manager: %v", err)
		return
	}
	err = dbManager.CreateTablesIfNeed()
	if err != nil {
		log.Fatalf("Can not init DB manager: %v", err)
		return
	}
	hs := Handlers{dbManager: &dbManager}

	// Initialize HTTP server
	mux := http.NewServeMux()

	mux.HandleFunc("POST /trades", hs.HandlePostTrades)
	mux.HandleFunc("GET /stats/{acc}", hs.HandleGetStats)
	mux.HandleFunc("GET /healthz", hs.HandleGetHealth)

	// Start server
	serverAddr := fmt.Sprintf(":%s", *listenAddr)
	log.Printf("Starting server on %s", serverAddr)
	if err = http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

type Handlers struct {
	dbManager *dbmanager.Manager
}

func (h *Handlers) HandleGetHealth(w http.ResponseWriter, r *http.Request) {
	// 1. Check database connection
	// 2. Return health status

	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}
	err := h.dbManager.Ping()
	if err != nil {
		if errors.Is(err, sql.ErrConnDone) {
			log.Print("Connection closed")
		} else {
			log.Print(err.Error())
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("StatusOK"))

}

func (h *Handlers) HandlePostTrades(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	trade := model.Trade{}

	err := json.NewDecoder(r.Body).Decode(&trade)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "invalid trade data", http.StatusBadRequest)
		return
	}

	if err = ValidateTrade(&trade); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.dbManager.CreateTrade(&trade)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "cant create new trade data", http.StatusInternalServerError)
		return
	}

	// TODO: Write code here
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandleGetStats(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	re, _ := regexp.Compile(`\/{[a-zA-Z0-9]+}$`)
	accountNo := re.FindString(r.URL.Path)

	if accountNo == "" {

		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	account, err := h.dbManager.GetClient(accountNo)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "cant get account data", http.StatusInternalServerError)
		return
	}
	if account == nil {
		http.Error(w, "account not found", http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(account)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "invalid response", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "cant create response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func ValidateTrade(t *model.Trade) error {
	return validator.New().Struct(t)
}
