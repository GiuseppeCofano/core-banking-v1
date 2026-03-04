package main

import "github.com/gcofano/core-banking-v1/models"

// Store is the abstraction over the ledger's persistence layer.
// Both the SQLite and Spanner backends implement this interface.
type Store interface {
	// CreateAccount inserts a new account and returns it.
	CreateAccount(owner, currency string) (*models.Account, error)

	// GetAccount retrieves an account by ID.
	GetAccount(id string) (*models.Account, error)

	// CreateLedgerEntry records an entry and updates the account balance atomically.
	CreateLedgerEntry(req models.CreateLedgerEntryRequest) (*models.LedgerEntry, error)

	// GetEntriesByAccount returns all ledger entries for a given account.
	GetEntriesByAccount(accountID string) ([]models.LedgerEntry, error)

	// Close releases any resources held by the store.
	Close() error
}
