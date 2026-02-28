package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	ledgerURL := os.Getenv("LEDGER_URL")
	if ledgerURL == "" {
		ledgerURL = "http://localhost:8080"
	}
	processorURL := os.Getenv("PROCESSOR_URL")
	if processorURL == "" {
		processorURL = "http://localhost:8082"
	}

	mux := http.NewServeMux()

	// Serve static files
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	// Proxy API calls to backend services
	mux.HandleFunc("/api/accounts", proxyTo(ledgerURL))
	mux.HandleFunc("/api/accounts/", proxyTo(ledgerURL))
	mux.HandleFunc("/api/ledger/", proxyTo(ledgerURL))
	mux.HandleFunc("/api/process/", proxyTo(processorURL))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("WebApp listening on :%s (ledger=%s, processor=%s)", port, ledgerURL, processorURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// proxyTo creates a handler that proxies requests to the target service.
// /api/accounts/... → targetURL/accounts/...
func proxyTo(targetURL string) http.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("invalid proxy target: %v", err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			// Strip /api prefix
			if len(req.URL.Path) > 4 && req.URL.Path[:4] == "/api" {
				req.URL.Path = req.URL.Path[4:]
			}
			req.Host = target.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, `{"error":"service unavailable"}`)
		},
	}
	return proxy.ServeHTTP
}
