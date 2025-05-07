package main

import (
	"database/sql"
	"encoding/json"
	"gitlab.com/digineat/go-broker-test/internal/db"
	"gitlab.com/digineat/go-broker-test/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// создаёт и инициализирует БД в памяти
func setupDB(t *testing.T) *sql.DB {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.InitDB(conn); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	return conn
}

// проверяем модель
func TestTradeInput_Validate(t *testing.T) {
	cases := []struct {
		in   model.TradeInput
		want bool
	}{
		{model.TradeInput{"a", "ABCDEF", 1, 1, 2, "buy"}, true},
		{model.TradeInput{"", "ABCDEF", 1, 1, 2, "buy"}, false},
		{model.TradeInput{"a", "ABCDE", 1, 1, 2, "buy"}, false},
		{model.TradeInput{"a", "ABCDEF", 0, 1, 2, "buy"}, false},
		{model.TradeInput{"a", "ABCDEF", 1, 1, 2, "hold"}, false},
	}
	for _, c := range cases {
		got := c.in.Validate()
		if got != c.want {
			t.Errorf("Validate(%#v) = %v; want %v", c.in, got, c.want)
		}
	}
}

// Проверяем HTTP POST /trades
func TestPostTradesHandler(t *testing.T) {
	conn := setupDB(t)
	handler := postTradesHandler(conn)

	// неверный метод
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trades", nil)
	handler(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /trades status = %d; want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	// некорректный JSON
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trades", strings.NewReader(`{}`))
	handler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("POST /trades invalid JSON status = %d; want %d", rr.Code, http.StatusBadRequest)
	}

	// корректный запрос
	body := `{"account":"a","symbol":"ABCDEF","volume":1,"open":1,"close":2,"side":"buy"}`
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/trades", strings.NewReader(body))
	handler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("POST /trades valid status = %d; want %d", rr.Code, http.StatusOK)
	}
	// убедимся, что сделка попала в очередь
	var cnt int
	if err := conn.QueryRow("SELECT count(*) FROM trades_q").Scan(&cnt); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if cnt != 1 {
		t.Errorf("expected 1 queued trade; got %d", cnt)
	}
}

// Проверяем HTTP GET /stats/{account}
func TestGetStatsHandler(t *testing.T) {
	conn := setupDB(t)
	handler := getStatsHandler(conn)

	// без указания account
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats/", nil)
	handler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("GET /stats/ status = %d; want %d", rr.Code, http.StatusBadRequest)
	}

	// для несуществующего аккаунта должно быть 0/0
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/stats/foo", nil)
	handler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET /stats/foo status = %d; want %d", rr.Code, http.StatusOK)
	}
	var statsResp db.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("unmarshal stats: %v", err)
	}
	if statsResp.Account != "foo" || statsResp.Trades != 0 || statsResp.Profit != 0 {
		t.Errorf("unexpected stats %+v; want {Account:foo Trades:0 Profit:0}", statsResp)
	}
}

// Тестим db.EnqueueTrade и db.GetStats напрямую
func TestEnqueueAndStatsDirect(t *testing.T) {
	conn := setupDB(t)
	ti := model.TradeInput{"x", "ABCDEF", 1, 1, 2, "buy"}
	if !ti.Validate() {
		t.Fatal("input did not validate")
	}
	if err := db.EnqueueTrade(conn, ti); err != nil {
		t.Fatalf("EnqueueTrade failed: %v", err)
	}
	// симулируем обработку вручную
	if err := db.ProcessNext(conn); err != nil {
		t.Fatalf("ProcessNext failed: %v", err)
	}
	stats, err := db.GetStats(conn, "x")
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.Trades != 1 || stats.Profit != (2-1)*1*100000.0 {
		t.Errorf("unexpected stats %+v", stats)
	}
}

func TestHealthHandler_OK(t *testing.T) {
	// Предварительно создаём и инициализируем БД
	conn := setupDB(t)
	handler := healthHandler(conn)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("healthHandler: status = %d; want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); body != "OK" {
		t.Errorf("healthHandler: body = %q; want \"OK\"", body)
	}
}

func TestHealthHandler_DBError(t *testing.T) {
	// Закрываем соединение, чтобы Ping() вернул ошибку
	conn := setupDB(t)
	conn.Close()

	handler := healthHandler(conn)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("healthHandler on closed DB: status = %d; want %d", rr.Code, http.StatusInternalServerError)
	}
}
