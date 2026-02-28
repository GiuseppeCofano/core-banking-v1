package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gcofano/core-banking-v1/models"
	"github.com/google/uuid"
)

// TransferSaga orchestrates a transfer using the Saga pattern.
// It runs: validate → debit source → credit destination.
// If the credit step fails, it compensates by reversing the debit.
type TransferSaga struct {
	svc   *BankingService
	txnID string
	req   models.TransferRequest
	steps []models.SagaStep
}

// NewTransferSaga creates a new saga for the given transfer request.
func NewTransferSaga(svc *BankingService, req models.TransferRequest) *TransferSaga {
	return &TransferSaga{
		svc:   svc,
		txnID: uuid.New().String(),
		req:   req,
		steps: make([]models.SagaStep, 0, 4),
	}
}

// Execute runs the saga steps and returns the transaction response.
func (s *TransferSaga) Execute() (*models.TransactionResponse, error) {
	// Step 1: Validate accounts and funds.
	if err := s.stepValidate(); err != nil {
		return s.result(models.TransactionStatusFailed, err.Error()), err
	}

	// Step 2: Debit the source account.
	if err := s.stepDebit(); err != nil {
		return s.result(models.TransactionStatusFailed, "debit failed: "+err.Error()), err
	}

	// Step 3: Credit the destination account.
	if err := s.stepCredit(); err != nil {
		log.Printf("SAGA %s: credit failed, starting compensation: %v", s.txnID, err)

		// Step 4: Compensate — reverse the debit.
		if compErr := s.stepCompensate(); compErr != nil {
			// Compensation also failed — critical situation.
			log.Printf("SAGA %s: CRITICAL — compensation also failed: %v", s.txnID, compErr)
			msg := fmt.Sprintf("credit failed (%v) AND compensation failed (%v) — manual intervention required", err, compErr)
			return s.result(models.TransactionStatusFailed, msg), fmt.Errorf(msg)
		}

		msg := fmt.Sprintf("credit failed (%v), debit reversed successfully", err)
		return s.result(models.TransactionStatusCompensated, msg), fmt.Errorf("transfer compensated: %w", err)
	}

	return s.result(
		models.TransactionStatusCompleted,
		fmt.Sprintf("Transferred %.2f from %s to %s", s.req.Amount, s.req.FromAccountID, s.req.ToAccountID),
	), nil
}

// --- Saga steps ---

func (s *TransferSaga) stepValidate() error {
	s.addStep("VALIDATE", models.SagaStepRunning, "")

	from, err := s.svc.getAccount(s.req.FromAccountID)
	if err != nil {
		s.failStep("VALIDATE", "source account not found: "+err.Error())
		return fmt.Errorf("source account not found: %w", err)
	}

	_, err = s.svc.getAccount(s.req.ToAccountID)
	if err != nil {
		s.failStep("VALIDATE", "destination account not found: "+err.Error())
		return fmt.Errorf("destination account not found: %w", err)
	}

	if from.Balance < s.req.Amount {
		msg := fmt.Sprintf("insufficient funds: available %.2f, requested %.2f", from.Balance, s.req.Amount)
		s.failStep("VALIDATE", msg)
		return fmt.Errorf(msg)
	}

	s.doneStep("VALIDATE")
	return nil
}

func (s *TransferSaga) stepDebit() error {
	s.addStep("DEBIT", models.SagaStepRunning, "")

	debitReq := models.CreateLedgerEntryRequest{
		TransactionID: s.txnID,
		AccountID:     s.req.FromAccountID,
		Type:          models.TransactionTypeTransfer,
		Amount:        -s.req.Amount,
		Description:   fmt.Sprintf("Transfer to %s: -%.2f", s.req.ToAccountID, s.req.Amount),
	}
	if err := s.svc.createLedgerEntry(debitReq); err != nil {
		s.failStep("DEBIT", err.Error())
		return err
	}

	s.doneStep("DEBIT")
	return nil
}

func (s *TransferSaga) stepCredit() error {
	s.addStep("CREDIT", models.SagaStepRunning, "")

	creditReq := models.CreateLedgerEntryRequest{
		TransactionID: s.txnID,
		AccountID:     s.req.ToAccountID,
		Type:          models.TransactionTypeTransfer,
		Amount:        s.req.Amount,
		Description:   fmt.Sprintf("Transfer from %s: +%.2f", s.req.FromAccountID, s.req.Amount),
	}
	if err := s.svc.createLedgerEntry(creditReq); err != nil {
		s.failStep("CREDIT", err.Error())
		return err
	}

	s.doneStep("CREDIT")
	return nil
}

func (s *TransferSaga) stepCompensate() error {
	s.addStep("COMPENSATE_DEBIT", models.SagaStepRunning, "")

	compensateReq := models.CreateLedgerEntryRequest{
		TransactionID: s.txnID,
		AccountID:     s.req.FromAccountID,
		Type:          models.TransactionTypeTransfer,
		Amount:        s.req.Amount, // positive: reverse the debit
		Description:   fmt.Sprintf("COMPENSATION: reversed debit of %.2f (credit to %s failed)", s.req.Amount, s.req.ToAccountID),
	}
	if err := s.svc.createLedgerEntry(compensateReq); err != nil {
		s.failStep("COMPENSATE_DEBIT", err.Error())
		return err
	}

	s.doneStep("COMPENSATE_DEBIT")
	return nil
}

// --- Step tracking helpers ---

func (s *TransferSaga) addStep(name string, status models.SagaStepStatus, errMsg string) {
	s.steps = append(s.steps, models.SagaStep{
		Name:      name,
		Status:    status,
		Error:     errMsg,
		Timestamp: time.Now().UTC(),
	})
}

func (s *TransferSaga) doneStep(name string) {
	s.updateLastStep(name, models.SagaStepDone, "")
}

func (s *TransferSaga) failStep(name string, errMsg string) {
	s.updateLastStep(name, models.SagaStepFailed, errMsg)
}

func (s *TransferSaga) updateLastStep(name string, status models.SagaStepStatus, errMsg string) {
	for i := len(s.steps) - 1; i >= 0; i-- {
		if s.steps[i].Name == name {
			s.steps[i].Status = status
			s.steps[i].Error = errMsg
			s.steps[i].Timestamp = time.Now().UTC()
			return
		}
	}
}

func (s *TransferSaga) result(status models.TransactionStatus, message string) *models.TransactionResponse {
	return &models.TransactionResponse{
		TransactionID: s.txnID,
		Status:        status,
		Message:       message,
		SagaSteps:     s.steps,
	}
}
