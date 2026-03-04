package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	backend := os.Getenv("DB_BACKEND")
	if backend == "" {
		backend = "sqlite"
	}

	var store Store
	var err error

	switch backend {
	case "spanner":
		spannerDB := os.Getenv("SPANNER_DATABASE")
		if spannerDB == "" {
			log.Fatal("SPANNER_DATABASE env var is required when DB_BACKEND=spanner " +
				"(format: projects/<project>/instances/<instance>/databases/<database>)")
		}
		store, err = NewSpannerStore(spannerDB)
		if err != nil {
			log.Fatalf("init spanner store: %v", err)
		}
		log.Printf("Using Spanner backend: %s", spannerDB)

	case "sqlite":
		dbDir := os.Getenv("DATA_DIR")
		if dbDir == "" {
			dbDir = "./data"
		}
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatalf("create data dir: %v", err)
		}
		dbPath := filepath.Join(dbDir, "banking.db")
		store, err = NewSQLiteStore(dbPath)
		if err != nil {
			log.Fatalf("init sqlite store: %v", err)
		}
		log.Printf("Using SQLite backend: %s", dbPath)

	default:
		log.Fatalf("unknown DB_BACKEND %q (supported: sqlite, spanner)", backend)
	}
	defer store.Close()

	h := NewHandlers(store)

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
