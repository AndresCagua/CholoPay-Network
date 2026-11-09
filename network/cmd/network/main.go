package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

type healthResponse struct { Status string `json:"status"` }

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "UP"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "UP"})
	})
	port := os.Getenv("SERVER_PORT")
	if port == "" { port = "8080" }
	server := &http.Server{Addr: ":" + port, Handler: mux}
	slog.Info("cholonet skeleton listening", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed { slog.Error("server stopped", "error", err); os.Exit(1) }
}
