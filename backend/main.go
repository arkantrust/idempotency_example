// Command backend starts the idempotency-example HTTP server.
//
// This project demonstrates idempotent REST API design using Go and BoltDB.
// Run with:
//
//	go run ./main.go
//
// The server listens on :8080 by default. Set the PORT environment variable
// to override. Set DB_PATH to change the BoltDB file location (default:
// chargebacks.db).
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/arkantrust/idempotency-example/backend/handlers"
	"github.com/arkantrust/idempotency-example/backend/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "chargebacks.db"
	}

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	h := handlers.New(s)

	mux := http.NewServeMux()

	// CORS middleware wraps every route so the React frontend (served on a
	// different port during development) can reach the API.
	mux.Handle("GET /chargebacks", corsMiddleware(http.HandlerFunc(h.ServeHTTP)))
	mux.Handle("POST /chargebacks/{id}", corsMiddleware(http.HandlerFunc(h.ServeHTTP)))
	mux.Handle("PUT /chargebacks/{id}", corsMiddleware(http.HandlerFunc(h.ServeHTTP)))
	mux.Handle("DELETE /chargebacks/{id}", corsMiddleware(http.HandlerFunc(h.ServeHTTP)))

	// Handle pre-flight OPTIONS requests for all paths.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			setCORSHeaders(w)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})

	log.Printf("listening on :%s (db: %s)", port, dbPath)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// setCORSHeaders adds CORS headers to a response.
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "X-Idempotency-Write")
}

// corsMiddleware wraps an http.Handler with CORS support.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
