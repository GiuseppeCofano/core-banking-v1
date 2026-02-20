package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	ledgerURL := os.Getenv("LEDGER_URL")
	if ledgerURL == "" {
		ledgerURL = "http://localhost:8080"
	}

	svc := NewBankingService(ledgerURL)
	h := NewCoreHandlers(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/deposit", h.HandleDeposit)
	mux.HandleFunc("/transfer", h.HandleTransfer)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Core service listening on :%s (ledger=%s)", port, ledgerURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
