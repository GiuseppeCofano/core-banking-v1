package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gcofano/core-banking-v1/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore wraps the SQLite connection and implements the Store interface.
type SQLiteStore struct {
	conn *sql.DB
}

// Compile-time check: SQLiteStore must satisfy Store.
var _ Store = (*SQLiteStore)(nil)

// NewSQLiteStore opens (or creates) the SQLite database and runs migrations.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db := &SQLiteStore{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS accounts (
		id         TEXT PRIMARY KEY,
		owner      TEXT NOT NULL,
		currency   TEXT NOT NULL DEFAULT 'EUR',
		balance    REAL NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS ledger_entries (
		id              TEXT PRIMARY KEY,
		transaction_id  TEXT NOT NULL,
		account_id      TEXT NOT NULL,
		type            TEXT NOT NULL,
		amount          REAL NOT NULL,
		balance         REAL NOT NULL,
		description     TEXT,
		created_at      DATETIME NOT NULL,
		FOREIGN KEY (account_id) REFERENCES accounts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_ledger_entries_account ON ledger_entries(account_id);
	CREATE INDEX IF NOT EXISTS idx_ledger_entries_transaction ON ledger_entries(transaction_id);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// CreateAccount inserts a new account and returns it.
func (db *SQLiteStore) CreateAccount(owner, currency string) (*models.Account, error) {
	now := time.Now().UTC()
	account := &models.Account{
		ID:        uuid.New().String(),
		Owner:     owner,
		Currency:  currency,
		Balance:   0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.conn.Exec(
		`INSERT INTO accounts (id, owner, currency, balance, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		account.ID, account.Owner, account.Currency, account.Balance,
		account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert account: %w", err)
	}
	return account, nil
}

// GetAccount retrieves an account by ID.
func (db *SQLiteStore) GetAccount(id string) (*models.Account, error) {
	row := db.conn.QueryRow(
		`SELECT id, owner, currency, balance, created_at, updated_at
		 FROM accounts WHERE id = ?`, id,
	)
	var a models.Account
	err := row.Scan(&a.ID, &a.Owner, &a.Currency, &a.Balance, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return &a, nil
}

// CreateLedgerEntry records an entry and updates the account balance atomically.
func (db *SQLiteStore) CreateLedgerEntry(req models.CreateLedgerEntryRequest) (*models.LedgerEntry, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Lock and read current balance.
	var currentBalance float64
	err = tx.QueryRow(`SELECT balance FROM accounts WHERE id = ?`, req.AccountID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("read balance: %w", err)
	}

	newBalance := currentBalance + req.Amount
	if newBalance < 0 {
		return nil, fmt.Errorf("insufficient funds: balance would be %.2f", newBalance)
	}

	now := time.Now().UTC()
	entry := &models.LedgerEntry{
		ID:            uuid.New().String(),
		TransactionID: req.TransactionID,
		AccountID:     req.AccountID,
		Type:          req.Type,
		Amount:        req.Amount,
		Balance:       newBalance,
		Description:   req.Description,
		CreatedAt:     now,
	}

	_, err = tx.Exec(
		`INSERT INTO ledger_entries (id, transaction_id, account_id, type, amount, balance, description, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.TransactionID, entry.AccountID, entry.Type,
		entry.Amount, entry.Balance, entry.Description, entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert entry: %w", err)
	}

	_, err = tx.Exec(
		`UPDATE accounts SET balance = ?, updated_at = ? WHERE id = ?`,
		newBalance, now, req.AccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return entry, nil
}

// GetEntriesByAccount returns all ledger entries for a given account.
func (db *SQLiteStore) GetEntriesByAccount(accountID string) ([]models.LedgerEntry, error) {
	rows, err := db.conn.Query(
		`SELECT id, transaction_id, account_id, type, amount, balance, description, created_at
		 FROM ledger_entries WHERE account_id = ? ORDER BY created_at ASC`, accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var e models.LedgerEntry
		if err := rows.Scan(&e.ID, &e.TransactionID, &e.AccountID, &e.Type,
			&e.Amount, &e.Balance, &e.Description, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Close closes the database connection.
func (db *SQLiteStore) Close() error {
	return db.conn.Close()
}
