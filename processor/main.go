package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	coreURL := os.Getenv("CORE_URL")
	if coreURL == "" {
		coreURL = "http://localhost:8081"
	}

	proc := NewProcessor(coreURL)
	h := NewProcessorHandlers(proc)

	mux := http.NewServeMux()
	mux.HandleFunc("/process/deposit", h.HandleProcessDeposit)
	mux.HandleFunc("/process/transfer", h.HandleProcessTransfer)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Processor service listening on :%s (core=%s)", port, coreURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
