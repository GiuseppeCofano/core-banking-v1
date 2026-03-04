package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gcofano/core-banking-v1/models"
)

// Handlers holds the database reference and exposes HTTP handler methods.
type Handlers struct {
	db Store
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db Store) *Handlers {
	return &Handlers{db: db}
}

// --- Accounts ---

// CreateAccount handles POST /accounts
func (h *Handlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "owner is required")
		return
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}

	account, err := h.db.CreateAccount(req.Owner, req.Currency)
	if err != nil {
		log.Printf("ERROR creating account: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

// GetAccount handles GET /accounts/{id}
func (h *Handlers) GetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/accounts/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	account, err := h.db.GetAccount(id)
	if err != nil {
		log.Printf("ERROR getting account %s: %v", id, err)
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, account)
}

// AccountsRouter routes /accounts requests.
func (h *Handlers) AccountsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/accounts")
	if path == "" || path == "/" {
		h.CreateAccount(w, r)
		return
	}
	// /accounts/{id}
	h.GetAccount(w, r)
}

// --- Ledger Entries ---

// CreateEntry handles POST /ledger/entries
func (h *Handlers) CreateEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.CreateLedgerEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.AccountID == "" || req.TransactionID == "" {
		writeError(w, http.StatusBadRequest, "account_id and transaction_id are required")
		return
	}

	entry, err := h.db.CreateLedgerEntry(req)
	if err != nil {
		log.Printf("ERROR creating entry: %v", err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

// GetEntries handles GET /ledger/entries/{account_id}
func (h *Handlers) GetEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	accountID := strings.TrimPrefix(r.URL.Path, "/ledger/entries/")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	entries, err := h.db.GetEntriesByAccount(accountID)
	if err != nil {
		log.Printf("ERROR getting entries: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve entries")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// EntriesRouter routes /ledger/entries requests.
func (h *Handlers) EntriesRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/ledger/entries")
	if path == "" || path == "/" {
		h.CreateEntry(w, r)
		return
	}
	// /ledger/entries/{account_id}
	h.GetEntries(w, r)
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
