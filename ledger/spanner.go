package main

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/gcofano/core-banking-v1/models"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// SpannerStore implements the Store interface using Google Cloud Spanner.
type SpannerStore struct {
	client *spanner.Client
}

// Compile-time check: SpannerStore must satisfy Store.
var _ Store = (*SpannerStore)(nil)

// NewSpannerStore creates a new SpannerStore.
// database should be the full resource path:
//
//	projects/<project>/instances/<instance>/databases/<database>
func NewSpannerStore(database string) (*SpannerStore, error) {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("create spanner client: %w", err)
	}
	return &SpannerStore{client: client}, nil
}

// CreateAccount inserts a new account and returns it.
func (s *SpannerStore) CreateAccount(owner, currency string) (*models.Account, error) {
	now := time.Now().UTC()
	account := &models.Account{
		ID:        uuid.New().String(),
		Owner:     owner,
		Currency:  currency,
		Balance:   0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.client.Apply(context.Background(), []*spanner.Mutation{
		spanner.InsertMap("accounts", map[string]interface{}{
			"id":         account.ID,
			"owner":      account.Owner,
			"currency":   account.Currency,
			"balance":    account.Balance,
			"created_at": account.CreatedAt,
			"updated_at": account.UpdatedAt,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("insert account: %w", err)
	}
	return account, nil
}

// GetAccount retrieves an account by ID.
func (s *SpannerStore) GetAccount(id string) (*models.Account, error) {
	row, err := s.client.Single().ReadRow(
		context.Background(),
		"accounts",
		spanner.Key{id},
		[]string{"id", "owner", "currency", "balance", "created_at", "updated_at"},
	)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	var a models.Account
	if err := row.Columns(&a.ID, &a.Owner, &a.Currency, &a.Balance, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan account: %w", err)
	}
	return &a, nil
}

// CreateLedgerEntry records an entry and updates the account balance atomically
// using a Spanner read-write transaction.
func (s *SpannerStore) CreateLedgerEntry(req models.CreateLedgerEntryRequest) (*models.LedgerEntry, error) {
	var entry *models.LedgerEntry

	_, err := s.client.ReadWriteTransaction(context.Background(),
		func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read current balance.
			row, err := txn.ReadRow(ctx, "accounts", spanner.Key{req.AccountID}, []string{"balance"})
			if err != nil {
				return fmt.Errorf("read balance: %w", err)
			}
			var currentBalance float64
			if err := row.Columns(&currentBalance); err != nil {
				return fmt.Errorf("scan balance: %w", err)
			}

			newBalance := currentBalance + req.Amount
			if newBalance < 0 {
				return fmt.Errorf("insufficient funds: balance would be %.2f", newBalance)
			}

			now := time.Now().UTC()
			entry = &models.LedgerEntry{
				ID:            uuid.New().String(),
				TransactionID: req.TransactionID,
				AccountID:     req.AccountID,
				Type:          req.Type,
				Amount:        req.Amount,
				Balance:       newBalance,
				Description:   req.Description,
				CreatedAt:     now,
			}

			return txn.BufferWrite([]*spanner.Mutation{
				// Insert the ledger entry.
				spanner.InsertMap("ledger_entries", map[string]interface{}{
					"id":             entry.ID,
					"transaction_id": entry.TransactionID,
					"account_id":     entry.AccountID,
					"type":           string(entry.Type),
					"amount":         entry.Amount,
					"balance":        entry.Balance,
					"description":    entry.Description,
					"created_at":     entry.CreatedAt,
				}),
				// Update account balance.
				spanner.UpdateMap("accounts", map[string]interface{}{
					"id":         req.AccountID,
					"balance":    newBalance,
					"updated_at": now,
				}),
			})
		})
	if err != nil {
		return nil, fmt.Errorf("spanner transaction: %w", err)
	}
	return entry, nil
}

// GetEntriesByAccount returns all ledger entries for a given account, ordered by created_at.
func (s *SpannerStore) GetEntriesByAccount(accountID string) ([]models.LedgerEntry, error) {
	stmt := spanner.Statement{
		SQL: `SELECT id, transaction_id, account_id, type, amount, balance, description, created_at
		      FROM ledger_entries
		      WHERE account_id = @accountID
		      ORDER BY created_at ASC`,
		Params: map[string]interface{}{
			"accountID": accountID,
		},
	}

	iter := s.client.Single().Query(context.Background(), stmt)
	defer iter.Stop()

	var entries []models.LedgerEntry
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query entries: %w", err)
		}

		var e models.LedgerEntry
		var entryType string
		if err := row.Columns(&e.ID, &e.TransactionID, &e.AccountID, &entryType,
			&e.Amount, &e.Balance, &e.Description, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		e.Type = models.TransactionType(entryType)
		entries = append(entries, e)
	}
	return entries, nil
}

// Close closes the Spanner client connection.
func (s *SpannerStore) Close() error {
	s.client.Close()
	return nil
}
