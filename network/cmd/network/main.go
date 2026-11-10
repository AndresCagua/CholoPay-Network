package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cholopay/network/internal/directory"
	"github.com/cholopay/network/internal/instruction"
	"github.com/cholopay/network/internal/message"
	"github.com/cholopay/network/internal/orchestration"
	"github.com/cholopay/network/internal/routing"
	"github.com/cholopay/network/internal/settlement"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type healthResponse struct {
	Status string `json:"status"`
}

type operatorOverview struct {
	SchemaVersion        string `json:"schema_version"`
	Source               string `json:"source"`
	LastSynchronizedAt   string `json:"last_synchronized_at"`
	ReconciliationStatus string `json:"reconciliation_status"`
	Networks             int    `json:"networks"`
	Participants         int    `json:"participants"`
	ActiveKeys           int    `json:"active_keys"`
	SuspendedKeys        int    `json:"suspended_keys"`
}

func operatorOverviewHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorizedBearer(r, os.Getenv("OPERATOR_CONSOLE_TOKEN")) {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}
		var overview operatorOverview
		overview.SchemaVersion = "1.0"
		overview.Source = "cholonet-network-read-model"
		overview.ReconciliationStatus = "UNKNOWN"
		if err := db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM payment_network`).Scan(&overview.Networks); err != nil {
			http.Error(w, "operator projection unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM network_participant`).Scan(&overview.Participants); err != nil {
			http.Error(w, "operator projection unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM directory_key WHERE status='ACTIVE'`).Scan(&overview.ActiveKeys); err != nil {
			http.Error(w, "operator projection unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM directory_key WHERE status='SUSPENDED'`).Scan(&overview.SuspendedKeys); err != nil {
			http.Error(w, "operator projection unavailable", http.StatusServiceUnavailable)
			return
		}
		overview.LastSynchronizedAt = time.Now().UTC().Format(time.RFC3339Nano)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(overview)
	}
}

func authorizedBearer(r *http.Request, expected string) bool {
	if expected == "" {
		return false
	}
	value := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return len(value) == len(expected) && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
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
	routingStore := routing.NewStore(db)
	if err := routingStore.Migrate(ctx); err != nil {
		slog.Error("migrate routing database", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	instruction.NewHandler(instruction.NewStore(db)).Register(mux)
	settlementStore := settlement.NewStore(db)
	settlement.NewHandler(settlementStore).Register(mux)
	settlement.NewReconciliationHandler(settlementStore).Register(mux)
	routing.NewHandler(routingStore).Register(mux)
	directoryStore := directory.NewStore(db)
	if err := directoryStore.Migrate(ctx); err != nil {
		slog.Error("migrate directory database", "error", err)
		os.Exit(1)
	}
	directory.NewHandler(directoryStore).Register(mux)
	messageStore := message.NewStore(db)
	if err := messageStore.Migrate(ctx); err != nil {
		slog.Error("migrate message database", "error", err)
		os.Exit(1)
	}
	message.NewHandler(messageStore).Register(mux)
	orchestration.NewHandler(orchestration.NewService(instruction.NewStore(db), routingStore, messageStore)).Register(mux)
	mux.HandleFunc("GET /internal/v1/operator/overview", operatorOverviewHandler(db))
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
