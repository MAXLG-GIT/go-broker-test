package main

import (
	"database/sql"
	dbmanager "gitlab.com/digineat/go-broker-test/internal/db"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

/*

HandleGetHealth
	передается не get запрос
	ping при закрытой базе

HandlePostTrades
	передается не post запрос
	передается кривой json, в том числе пустой
		Account   string  `json:"account" validate:"required,alphanum"`
		Symbol    string  `json:"symbol"  validate:"required,alpha,len=6"`
		Volume    float64 `json:"volume"  validate:"gt=0"`
		Open      float64 `json:"open"    validate:"gt=0"`
		Close     float64 `json:"close"   validate:"gt=0"`
		Side      string  `json:"side"    validate:"oneof=buy sell"`
HandleGetStats
	передается не get запрос
	кривой id аккаунта

*/

func Test_HandleGetHealth_Request(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
		disableDb  bool
	}{
		{name: "incorrect method", method: http.MethodGet, statusCode: http.StatusOK},
		{name: "correct method", method: http.MethodPost, statusCode: http.StatusMethodNotAllowed},
		{name: "db closed on execution", method: http.MethodGet, statusCode: http.StatusInternalServerError, disableDb: true},
	}

	var db *sql.DB
	dbManager := dbmanager.Manager{}
	hs := Handlers{dbManager: &dbManager}
	db = initDb()
	defer func(db *sql.DB) {
		closeDb(db)
		deleteDb()
	}(db)
	err := hs.dbManager.InitDbManager(db)
	if err != nil {
		t.Fatalf("ошибка инициализации бд: %s", err.Error())
		return
	}
	for _, test := range tests {
		t.Log(test.name)
		req := httptest.NewRequest(test.method, "/healthz", nil)
		wrec := httptest.NewRecorder()

		if test.disableDb == true {
			closeDb(db)
		}
		hs.HandleGetHealth(wrec, req)
		res := wrec.Result()
		defer res.Body.Close()

		if res.StatusCode != test.statusCode {
			t.Fatalf("ожидался статус %d, получили %d", test.statusCode, res.StatusCode)
		}
		t.Log("--Passed")
	}

}

func Test_HandlePostTrades(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
		disableDb  bool
		reqJson    string
	}{
		{name: "incorrect method", method: http.MethodGet, statusCode: http.StatusMethodNotAllowed},
		{name: "correct request", method: http.MethodPost,
			reqJson:    `{"account":"123","symbol":"EURUSD","volume":1.0,"open":1.1000,"close":1.1050,"side":"buy"}`,
			statusCode: http.StatusOK},
		{name: "empty json", method: http.MethodPost,
			reqJson:    ``,
			statusCode: http.StatusBadRequest},
		{name: "invalid json", method: http.MethodPost,
			reqJson:    `{"account":"123","symbol":"EURUSD","volume"<invalid_here>1.0,"open":1.1000,"close":1.1050,"side":"buy"}`,
			statusCode: http.StatusBadRequest},
		{name: "invalid json values", method: http.MethodPost,
			reqJson:    `{"account":"123","symbol":"FFF","volume":1.0,"open":1.1000,"close":no,"side":"buy"}`,
			statusCode: http.StatusBadRequest},
		{name: "db closed on execution", method: http.MethodPost, disableDb: true,
			reqJson:    `{"account":"123","symbol":"EURUSD","volume":1.0,"open":1.1000,"close":1.1050,"side":"buy"}`,
			statusCode: http.StatusInternalServerError},
	}

	var db *sql.DB
	dbManager := dbmanager.Manager{}
	hs := Handlers{dbManager: &dbManager}
	db = initDb()
	defer func(db *sql.DB) {
		closeDb(db)
		deleteDb()
	}(db)
	err := hs.dbManager.InitDbManager(db)
	if err != nil {
		t.Fatalf("ошибка инициализации бд: %s", err.Error())
		return
	}
	err = dbManager.CreateTablesIfNeed()
	if err != nil {
		t.Fatalf("ошибка создания таблиц бд: %s", err.Error())
	}
	for _, test := range tests {
		t.Log(test.name)
		req := httptest.NewRequest(test.method, "/trades", strings.NewReader(test.reqJson))
		wrec := httptest.NewRecorder()

		if test.disableDb == true {
			closeDb(db)
		}
		hs.HandlePostTrades(wrec, req)
		res := wrec.Result()
		defer res.Body.Close()

		if res.StatusCode != test.statusCode {
			t.Fatalf("ожидался статус %d, получили %d", test.statusCode, res.StatusCode)
		}
		t.Log("--Passed")
	}

}

func Test_HandleGetStats(t *testing.T) {
	//TODO implement
}

func initDb() *sql.DB {
	dbPath := "./../../data/data_test.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	return db
}

func deleteDb() {
	dbPath := "./../../data/data_test.db"
	err := os.Remove(dbPath)
	if err != nil {
		log.Fatalf("Cant delete test database: %v", err)

	}
}

func fillDbTableTradesQ() {

}

func closeDb(db *sql.DB) {
	err := db.Close()
	if err != nil {
		log.Fatalf("Failed to close database connection: %v", err)
	}
}
