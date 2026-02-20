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

// Processor validates incoming requests and forwards them to the Core service.
type Processor struct {
	coreURL    string
	httpClient *http.Client
}

// NewProcessor creates a new Processor.
func NewProcessor(coreURL string) *Processor {
	return &Processor{
		coreURL:    coreURL,
		httpClient: &http.Client{},
	}
}

// ProcessDeposit validates a deposit request and forwards to Core.
func (p *Processor) ProcessDeposit(req models.DepositRequest) (*models.TransactionResponse, error) {
	// Validate.
	if req.AccountID == "" {
		return nil, fmt.Errorf("account_id is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Generate idempotency key (for traceability).
	idempotencyKey := uuid.New().String()
	_ = idempotencyKey // Can be used in headers for future enhancement.

	// Forward to Core.
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.httpClient.Post(
		p.coreURL+"/deposit",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("call core service: %w", err)
	}
	defer resp.Body.Close()

	return p.parseResponse(resp)
}

// ProcessTransfer validates a transfer request and forwards to Core.
func (p *Processor) ProcessTransfer(req models.TransferRequest) (*models.TransactionResponse, error) {
	// Validate.
	if req.FromAccountID == "" {
		return nil, fmt.Errorf("from_account_id is required")
	}
	if req.ToAccountID == "" {
		return nil, fmt.Errorf("to_account_id is required")
	}
	if req.FromAccountID == req.ToAccountID {
		return nil, fmt.Errorf("from and to accounts must be different")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Forward to Core.
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.httpClient.Post(
		p.coreURL+"/transfer",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("call core service: %w", err)
	}
	defer resp.Body.Close()

	return p.parseResponse(resp)
}

func (p *Processor) parseResponse(resp *http.Response) (*models.TransactionResponse, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse as TransactionResponse first.
		var txnResp models.TransactionResponse
		if json.Unmarshal(respBody, &txnResp) == nil && txnResp.TransactionID != "" {
			return &txnResp, fmt.Errorf("core returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("core returned %d: %s", resp.StatusCode, string(respBody))
	}

	var txnResp models.TransactionResponse
	if err := json.Unmarshal(respBody, &txnResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &txnResp, nil
}
