package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gitlab.com/digineat/go-broker-test/internal/model"
	"log"
)

const Trades_table = "trades_q"
const Clients_table = "clients"

type Manager struct {
	db  *sql.DB
	ctx context.Context
}

//account	string	must not be empty
//symbol	string	^[A-Z]{6}$ (e.g. EURUSD)
//volume	float64	must be > 0
//open	float64	must be > 0
//close	float64	must be > 0
//side	string	either "buy" or "sell"

func (m *Manager) Ping() error {
	return m.db.Ping()
}

func (m *Manager) InitDbManager(db *sql.DB) error {
	if db == nil {
		return errors.New("no DB found")
	}
	m.db = db
	m.ctx = context.Background()

	return nil
}

func (m *Manager) CreateTablesIfNeed() error {
	err := m.CreateTradesQ()
	if err != nil {
		return errors.New(fmt.Sprintf("Can not create TradesQ table: %v", err))
	}

	err = m.CreateClients()
	if err != nil {
		return errors.New(fmt.Sprintf("Can not create Clients table: %v", err))
	}
	return nil
}

func (m *Manager) CreateTradesQ() error {
	schemaSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	account STRING UNIQUE,
	symbol VARCHAR(50),
	volume FLOAT,
    open FLOAT,
    close FLOAT,
    side VARCHAR(50),
    processed INTEGER DEFAULT(0)
);
`, Trades_table)
	if _, err := m.db.Exec(schemaSQL); err != nil {
		return err
	}
	return nil
}

func (m *Manager) CreateClients() error {
	schemaSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account STRING UNIQUE,
	trades INTEGER UNSIGNED,
	profit FLOAT,
	FOREIGN KEY (account)
       REFERENCES %s (account) 
);
`, Clients_table, Trades_table)
	if _, err := m.db.Exec(schemaSQL); err != nil {
		return err
	}
	return nil
}

func (m *Manager) CreateTrade(trade *model.Trade) error {

	tx, err := m.db.BeginTx(m.ctx, nil)
	if err != nil {
		return err
	}

	reqSQL := fmt.Sprintf(`
INSERT INTO %s (
    account, symbol, volume, open, close, side
) VALUES (
     ?, ?, ?, ?, ?, ?
 )
`, Trades_table)
	if _, err = tx.Exec(reqSQL,
		trade.Account, trade.Symbol, trade.Volume, trade.Open, trade.Close, trade.Side); err != nil {
		err = tx.Rollback()
		if err != nil {
			return err
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) GetClient(tradeNo string) (*model.Trade, error) {

	reqSQL := fmt.Sprintf(`
SELECT * FROM %s WHERE account = "?"
`, Trades_table)
	row := m.db.QueryRow(reqSQL, tradeNo)

	var trade model.Trade
	if err := row.Scan(&trade.Account, &trade.Symbol, &trade.Volume, &trade.Open, &trade.Close, &trade.Side); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println(err)
			return nil, nil
		}
		log.Fatal(err)
		return nil, err
	}

	return &trade, nil
}

func (m *Manager) GetTrade(ctx context.Context, tx *sql.Tx) (*model.Trade, error) {

	var trade model.Trade
	reqSQL := fmt.Sprintf(`
UPDATE %s
   SET processed = 1
 WHERE id = (
	 SELECT id
	   FROM trades_q
	  WHERE processed = 0
	  ORDER BY id
	  LIMIT 1
 )
RETURNING account, side, volume, open, close, processed;
`, Trades_table)
	err := tx.QueryRowContext(ctx, reqSQL).Scan(
		&trade.Account,
		&trade.Side,
		&trade.Volume,
		&trade.Open,
		&trade.Close,
		&trade.Processed,
	)

	return &trade, err
}

func (m *Manager) UpdateAccount(ctx context.Context, tx *sql.Tx, account string, profit float64) error {
	reqSQL := fmt.Sprintf(`
INSERT INTO %s(account, trades, profit) VALUES( ?, ?, ?)
ON CONFLICT(account) DO UPDATE SET trades = trades + ?, profit = profit + ?;`, Clients_table)
	_, err := tx.ExecContext(ctx, reqSQL, account, 1, profit, 1, profit)
	return err
}

//TODO export tx as interface

func (m *Manager) CreateTx(ctx context.Context) (*sql.Tx, error) {

	return m.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
}

func (m *Manager) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

func (m *Manager) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}
