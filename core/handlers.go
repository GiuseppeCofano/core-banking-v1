package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gcofano/core-banking-v1/models"
)

// CoreHandlers exposes HTTP handler methods for the Core service.
type CoreHandlers struct {
	svc *BankingService
}

// NewCoreHandlers creates a new CoreHandlers.
func NewCoreHandlers(svc *BankingService) *CoreHandlers {
	return &CoreHandlers{svc: svc}
}

// HandleDeposit handles POST /deposit
func (h *CoreHandlers) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	resp, err := h.svc.Deposit(req)
	if err != nil {
		log.Printf("ERROR deposit: %v", err)
		if resp != nil {
			writeJSON(w, http.StatusUnprocessableEntity, resp)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleTransfer handles POST /transfer
func (h *CoreHandlers) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	resp, err := h.svc.Transfer(req)
	if err != nil {
		log.Printf("ERROR transfer: %v", err)
		if resp != nil {
			writeJSON(w, http.StatusUnprocessableEntity, resp)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}
