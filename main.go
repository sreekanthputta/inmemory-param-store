// Parameter Store - a crash-safe, append-only key-value store
//
// All changes are appended to a JSONL file (never modified/deleted),
// providing a complete audit trail. An in-memory index enables fast lookups.
// Atomic batch writes with fsync ensure crash safety.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"parameter-store/internal/api"
	"parameter-store/internal/store"
)

//go:embed web
var webFS embed.FS

func main() {
	port := flag.Int("port", 8847, "HTTP server port")
	dataFile := flag.String("data", "data.jsonl", "Path to data file")
	flag.Parse()

	// Initialize store - this replays the log to rebuild the in-memory index
	paramStore, err := store.New(*dataFile)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	handler := api.NewHandler(paramStore)

	// API routes
	http.HandleFunc("/api/update", handler.Update)    // POST - batch update
	http.HandleFunc("/api/list", handler.List)        // GET - list all params
	http.HandleFunc("/api/get", handler.GetUnmasked)  // GET - single param unmasked
	http.HandleFunc("/api/history", handler.GetHistory) // GET - key change history
	http.HandleFunc("/api/health", handler.Health)    // GET - health check

	// Serve embedded web UI
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to get web content: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(webContent)))

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Data file: %s (%d parameters loaded)\n", *dataFile, paramStore.Count())

	// Check for TLS certificates
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	if certFile != "" && keyFile != "" {
		fmt.Printf("Parameter Store running on https://localhost%s\n", addr)
		if err := http.ListenAndServeTLS(addr, certFile, keyFile, nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	} else {
		fmt.Printf("Parameter Store running on http://localhost%s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
