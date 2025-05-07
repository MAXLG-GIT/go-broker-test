package main

import (
	"database/sql"
	"gitlab.com/digineat/go-broker-test/internal/db"
	"gitlab.com/digineat/go-broker-test/internal/model"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// создаём и инициализируем БД в памяти
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

// проверяем InitDB создание таблиц
func TestInitDB_CreatesTables(t *testing.T) {
	conn := setupDB(t)
	rows, err := conn.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		found[name] = true
	}
	for _, tbl := range []string{"trades_q", "account_stats"} {
		if !found[tbl] {
			t.Errorf("expected table %s to exist", tbl)
		}
	}
}

// проверяем ошибку при отсутствии сделок
func TestProcessNext_NoRows(t *testing.T) {
	conn := setupDB(t)
	err := db.ProcessNext(conn)
	if err != db.ErrNoTrade {
		t.Errorf("expected ErrNoTrade; got %v", err)
	}
}

// пробуем buy-ветку
func TestProcessNext_BuySide(t *testing.T) {
	conn := setupDB(t)
	// добавляем заявку
	ti := model.TradeInput{"b", "ABCDEF", 2, 10, 15, "buy"}
	if err := db.EnqueueTrade(conn, ti); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// обрабатываем
	if err := db.ProcessNext(conn); err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	stats, err := db.GetStats(conn, "b")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	expected := (15 - 10) * 2 * 100000.0
	if stats.Trades != 1 || stats.Profit != expected {
		t.Errorf("buy stats = %+v; want Trades=1 Profit=%v", stats, expected)
	}
}

// пробуем sell-ветку
func TestProcessNext_SellSide(t *testing.T) {
	conn := setupDB(t)
	ti := model.TradeInput{"s", "ABCDEF", 1, 20, 15, "sell"}
	if err := db.EnqueueTrade(conn, ti); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := db.ProcessNext(conn); err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	stats, err := db.GetStats(conn, "s")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	// для sell profit = -(close-open)*volume*100000
	expected := -(15 - 20) * 1 * 100000.0
	if stats.Trades != 1 || stats.Profit != expected {
		t.Errorf("sell stats = %+v; want Trades=1 Profit=%v", stats, expected)
	}
}

// несколько сделок подряд
func TestProcessMultipleTrades(t *testing.T) {
	conn := setupDB(t)
	batch := []model.TradeInput{
		{"m", "ABCDEF", 1, 1, 2, "buy"},
		{"m", "ABCDEF", 0.5, 2, 1.5, "sell"},
	}
	for _, ti := range batch {
		if err := db.EnqueueTrade(conn, ti); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	for i := range batch {
		if err := db.ProcessNext(conn); err != nil {
			t.Fatalf("ProcessNext #%d: %v", i, err)
		}
	}
	// больше нет
	if err := db.ProcessNext(conn); err != db.ErrNoTrade {
		t.Errorf("expected ErrNoTrade; got %v", err)
	}
	// итоговая статистика
	stats, err := db.GetStats(conn, "m")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	exp := (2-1)*1*100000.0 + -(1.5-2)*0.5*100000.0
	if stats.Trades != len(batch) || stats.Profit != exp {
		t.Errorf("final stats = %+v; want Trades=%d Profit=%v", stats, len(batch), exp)
	}
}
