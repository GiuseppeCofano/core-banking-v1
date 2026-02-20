package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gcofano/core-banking-v1/models"
	"github.com/google/uuid"
)

// BankingService contains the business logic for deposits and transfers.
type BankingService struct {
	ledgerURL  string
	httpClient *http.Client
}

// NewBankingService creates a new BankingService.
func NewBankingService(ledgerURL string) *BankingService {
	return &BankingService{
		ledgerURL:  ledgerURL,
		httpClient: &http.Client{},
	}
}

// Deposit credits an account with the given amount.
func (s *BankingService) Deposit(req models.DepositRequest) (*models.TransactionResponse, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive")
	}

	// Verify account exists.
	_, err := s.getAccount(req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	txnID := uuid.New().String()

	// Create a credit entry in the ledger.
	entryReq := models.CreateLedgerEntryRequest{
		TransactionID: txnID,
		AccountID:     req.AccountID,
		Type:          models.TransactionTypeDeposit,
		Amount:        req.Amount, // positive = credit
		Description:   fmt.Sprintf("Deposit of %.2f", req.Amount),
	}
	if err := s.createLedgerEntry(entryReq); err != nil {
		return &models.TransactionResponse{
			TransactionID: txnID,
			Status:        models.TransactionStatusFailed,
			Message:       err.Error(),
		}, err
	}

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        models.TransactionStatusCompleted,
		Message:       fmt.Sprintf("Deposited %.2f successfully", req.Amount),
	}, nil
}

// Transfer moves funds from one account to another using double-entry bookkeeping.
func (s *BankingService) Transfer(req models.TransferRequest) (*models.TransactionResponse, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}
	if req.FromAccountID == req.ToAccountID {
		return nil, fmt.Errorf("cannot transfer to the same account")
	}

	// Verify both accounts exist.
	from, err := s.getAccount(req.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("source account not found: %w", err)
	}
	_, err = s.getAccount(req.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("destination account not found: %w", err)
	}

	// Check sufficient funds.
	if from.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient funds: available %.2f, requested %.2f", from.Balance, req.Amount)
	}

	txnID := uuid.New().String()

	// Debit the source account.
	debitReq := models.CreateLedgerEntryRequest{
		TransactionID: txnID,
		AccountID:     req.FromAccountID,
		Type:          models.TransactionTypeTransfer,
		Amount:        -req.Amount, // negative = debit
		Description:   fmt.Sprintf("Transfer to %s: -%.2f", req.ToAccountID, req.Amount),
	}
	if err := s.createLedgerEntry(debitReq); err != nil {
		return &models.TransactionResponse{
			TransactionID: txnID,
			Status:        models.TransactionStatusFailed,
			Message:       "debit failed: " + err.Error(),
		}, err
	}

	// Credit the destination account.
	creditReq := models.CreateLedgerEntryRequest{
		TransactionID: txnID,
		AccountID:     req.ToAccountID,
		Type:          models.TransactionTypeTransfer,
		Amount:        req.Amount, // positive = credit
		Description:   fmt.Sprintf("Transfer from %s: +%.2f", req.FromAccountID, req.Amount),
	}
	if err := s.createLedgerEntry(creditReq); err != nil {
		return &models.TransactionResponse{
			TransactionID: txnID,
			Status:        models.TransactionStatusFailed,
			Message:       "credit failed: " + err.Error(),
		}, err
	}

	return &models.TransactionResponse{
		TransactionID: txnID,
		Status:        models.TransactionStatusCompleted,
		Message:       fmt.Sprintf("Transferred %.2f from %s to %s", req.Amount, req.FromAccountID, req.ToAccountID),
	}, nil
}

// --- Internal helpers ---

func (s *BankingService) getAccount(id string) (*models.Account, error) {
	resp, err := s.httpClient.Get(s.ledgerURL + "/accounts/" + id)
	if err != nil {
		return nil, fmt.Errorf("call ledger: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ledger returned %d: %s", resp.StatusCode, string(body))
	}

	var account models.Account
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("decode account: %w", err)
	}
	return &account, nil
}

func (s *BankingService) createLedgerEntry(req models.CreateLedgerEntryRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	resp, err := s.httpClient.Post(
		s.ledgerURL+"/ledger/entries",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("call ledger: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ledger returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
