package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gcofano/core-banking-v1/models"
)

// ProcessorHandlers exposes HTTP handler methods for the Processor service.
type ProcessorHandlers struct {
	proc *Processor
}

// NewProcessorHandlers creates a new ProcessorHandlers.
func NewProcessorHandlers(proc *Processor) *ProcessorHandlers {
	return &ProcessorHandlers{proc: proc}
}

// HandleProcessDeposit handles POST /process/deposit
func (h *ProcessorHandlers) HandleProcessDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	resp, err := h.proc.ProcessDeposit(req)
	if err != nil {
		log.Printf("ERROR process deposit: %v", err)
		if resp != nil {
			writeJSON(w, http.StatusUnprocessableEntity, resp)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleProcessTransfer handles POST /process/transfer
func (h *ProcessorHandlers) HandleProcessTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	resp, err := h.proc.ProcessTransfer(req)
	if err != nil {
		log.Printf("ERROR process transfer: %v", err)
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
