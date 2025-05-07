package db

import (
	"database/sql"
	"errors"
	_ "errors"
	"gitlab.com/digineat/go-broker-test/internal/model"
)

var ErrNoTrade = sql.ErrNoRows

func InitDB(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trades_q (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            account TEXT NOT NULL,
            symbol TEXT NOT NULL,
            volume REAL NOT NULL,
            open REAL NOT NULL,
            close REAL NOT NULL,
            side TEXT NOT NULL,
            processed INTEGER NOT NULL DEFAULT 0
        );`,
		`CREATE TABLE IF NOT EXISTS account_stats (
            account TEXT PRIMARY KEY,
            trades INTEGER NOT NULL DEFAULT 0,
            profit REAL NOT NULL DEFAULT 0
        );`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func EnqueueTrade(db *sql.DB, t model.TradeInput) error {
	_, err := db.Exec(
		`INSERT INTO trades_q(account,symbol,volume,open,close,side)
         VALUES(?,?,?,?,?,?)`,
		t.Account, t.Symbol, t.Volume, t.Open, t.Close, t.Side,
	)
	return err
}

type Stats struct {
	Account string  `json:"account"`
	Trades  int     `json:"trades"`
	Profit  float64 `json:"profit"`
}

func GetStats(db *sql.DB, account string) (Stats, error) {
	var s Stats
	s.Account = account
	err := db.QueryRow(
		`SELECT trades, profit FROM account_stats WHERE account = ?`, account,
	).Scan(&s.Trades, &s.Profit)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil // пустая статистика
	}
	return s, err
}

func ProcessNext(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var t model.Trade
	err = tx.QueryRow(
		`SELECT id, account, symbol, volume, open, close, side
         FROM trades_q WHERE processed = 0
         ORDER BY id LIMIT 1`,
	).Scan(&t.ID, &t.Account, &t.Symbol, &t.Volume, &t.Open, &t.Close, &t.Side)
	if err != nil {
		return err
	}

	lot := 100000.0
	profit := (t.Close - t.Open) * t.Volume * lot
	if t.Side == "sell" {
		profit = -profit
	}

	_, err = tx.Exec(
		`INSERT INTO account_stats(account,trades,profit)
         VALUES(?,?,?)
         ON CONFLICT(account) DO UPDATE
         SET trades = trades + 1,
             profit = profit + ?`,
		t.Account, 1, profit, profit,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE trades_q SET processed = 1 WHERE id = ?`, t.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
