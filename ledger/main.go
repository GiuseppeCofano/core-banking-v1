package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Database path — defaults to ./data/banking.db
	dbDir := os.Getenv("DATA_DIR")
	if dbDir == "" {
		dbDir = "./data"
	}
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "banking.db")

	db, err := NewDB(dbPath)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}
	defer db.Close()

	h := NewHandlers(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/", h.AccountsRouter)
	mux.HandleFunc("/accounts", h.AccountsRouter)
	mux.HandleFunc("/ledger/entries/", h.EntriesRouter)
	mux.HandleFunc("/ledger/entries", h.EntriesRouter)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Ledger service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
