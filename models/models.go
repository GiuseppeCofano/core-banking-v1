package models

import "time"

// TransactionType represents the type of transaction.
type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "DEPOSIT"
	TransactionTypeTransfer TransactionType = "TRANSFER"
)

// TransactionStatus represents the status of a transaction.
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// Account represents a bank account.
type Account struct {
	ID        string    `json:"id"`
	Owner     string    `json:"owner"`
	Currency  string    `json:"currency"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LedgerEntry represents a single entry in the double-entry ledger.
type LedgerEntry struct {
	ID            string          `json:"id"`
	TransactionID string          `json:"transaction_id"`
	AccountID     string          `json:"account_id"`
	Type          TransactionType `json:"type"`
	Amount        float64         `json:"amount"` // positive = credit, negative = debit
	Balance       float64         `json:"balance"` // balance after this entry
	Description   string          `json:"description"`
	CreatedAt     time.Time       `json:"created_at"`
}

// --- API Request / Response types ---

// CreateAccountRequest is the payload to create an account.
type CreateAccountRequest struct {
	Owner    string `json:"owner"`
	Currency string `json:"currency"`
}

// CreateLedgerEntryRequest is the payload to record a ledger entry.
type CreateLedgerEntryRequest struct {
	TransactionID string          `json:"transaction_id"`
	AccountID     string          `json:"account_id"`
	Type          TransactionType `json:"type"`
	Amount        float64         `json:"amount"`
	Description   string          `json:"description"`
}

// DepositRequest is the payload sent to Core / Processor to deposit funds.
type DepositRequest struct {
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
}

// TransferRequest is the payload sent to Core / Processor to transfer funds.
type TransferRequest struct {
	FromAccountID string  `json:"from_account_id"`
	ToAccountID   string  `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

// TransactionResponse is a generic response after processing a transaction.
type TransactionResponse struct {
	TransactionID string            `json:"transaction_id"`
	Status        TransactionStatus `json:"status"`
	Message       string            `json:"message,omitempty"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
