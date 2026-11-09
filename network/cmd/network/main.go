package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/settlement"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type healthResponse struct {
	Status string `json:"status"`
}

func main() {
	dbURL := os.Getenv("NETWORK_DB_URL")
	if dbURL == "" {
		slog.Error("NETWORK_DB_URL is required")
		os.Exit(1)
	}
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		slog.Error("open network database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		slog.Error("connect network database", "error", err)
		os.Exit(1)
	}
	if err := instruction.Migrate(ctx, db); err != nil {
		slog.Error("migrate network database", "error", err)
		os.Exit(1)
	}
	if err := settlement.Migrate(ctx, db); err != nil {
		slog.Error("migrate settlement database", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	instruction.NewHandler(instruction.NewStore(db)).Register(mux)
	settlementStore := settlement.NewStore(db)
	settlement.NewHandler(settlementStore).Register(mux)
	settlement.NewReconciliationHandler(settlementStore).Register(mux)
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "UP"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "UP"})
	})
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: mux}
	slog.Info("cholonet skeleton listening", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
