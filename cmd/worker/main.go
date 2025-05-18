package main

import (
	"context"
	"database/sql"
	"flag"
	dbmanager "gitlab.com/digineat/go-broker-test/internal/db"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Command line flags
	dbPath := flag.String("db", "data.db", "path to SQLite database")
	pollInterval := flag.Duration("poll", 100*time.Millisecond, "polling interval")
	flag.Parse()

	// Initialize database connection
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func(db *sql.DB) {
		err = db.Close()
		if err != nil {
			log.Fatalf("Failed to close database: %v", err)
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

	log.Printf("Worker started with polling interval: %v", *pollInterval)

	// Main worker loop
	lot := 100000.0
	for {
		// TODO: Write code here

		// при начале транзакции забирается строка из базы trade и помечается как FOR UPDATE

		// creating transaction
		ctx := context.Background()
		tx, txErr := dbManager.CreateTx(ctx)
		if txErr != nil {
			errRb := dbManager.RollbackTx(tx)
			if errRb != nil {
				log.Printf("Failed to rolback transaction")
				return
			}
			log.Printf("Не удалось получить trade: %v", err)
			return
		}

		trade, tradeErr := dbManager.GetTrade(ctx, tx)

		if tradeErr != nil {
			errRb := dbManager.RollbackTx(tx)
			if errRb != nil {
				log.Printf("Failed to rolback transaction")
				return
			}
			log.Printf("Не удалось получить trade: %v", tradeErr)
			return
		}
		if trade == nil {
			errRb := dbManager.RollbackTx(tx)
			if errRb != nil {
				log.Printf("Failed to rolback transaction")
				return
			}
			time.Sleep(*pollInterval)
			continue
		}

		profit := (trade.Close - trade.Open) * trade.Volume * lot
		if trade.Side == "sell" {
			profit = -profit
		}

		err = dbManager.UpdateAccount(ctx, tx, trade.Account, profit)
		if err != nil {
			errRb := dbManager.RollbackTx(tx)
			if errRb != nil {
				log.Printf("Failed to rolback transaction")
				return
			}
			log.Printf("Не удалось обновить аккаунт: %v", err)
			return
		}

		err = dbManager.CommitTx(tx)
		if err != nil {
			errRb := dbManager.RollbackTx(tx)
			if errRb != nil {
				log.Printf("Failed to rolback transaction")
				return
			}
			log.Printf("Не удалось обновить аккаунт: %v", err)
			return
		}

		// Sleep for the specified interval
		time.Sleep(*pollInterval)
	}
}
