package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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

type operatorIncident struct {
	Code   string `json:"code"`
	Count  int    `json:"count"`
	Source string `json:"source"`
}

type operatorHealth struct {
	NetworksTotal         int `json:"networks_total"`
	NetworksActive        int `json:"networks_active"`
	NetworksSuspended     int `json:"networks_suspended"`
	ParticipantsTotal     int `json:"participants_total"`
	ParticipantsActive    int `json:"participants_active"`
	ParticipantsSuspended int `json:"participants_suspended"`
	ActiveKeys            int `json:"active_keys"`
	SuspendedKeys         int `json:"suspended_keys"`
}

type operatorOverview struct {
	SchemaVersion        string             `json:"schema_version"`
	Source               string             `json:"source"`
	LastSynchronizedAt   string             `json:"last_synchronized_at"`
	ReconciliationStatus string             `json:"reconciliation_status"`
	Networks             int                `json:"networks"`
	Participants         int                `json:"participants"`
	ActiveKeys           int                `json:"active_keys"`
	SuspendedKeys        int                `json:"suspended_keys"`
	Health               operatorHealth     `json:"health"`
	Incidents            []operatorIncident `json:"incidents"`
}

func operatorOverviewHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorizedBearer(r, os.Getenv("OPERATOR_CONSOLE_TOKEN")) {
			http.Error(w, "se requiere autorización", http.StatusUnauthorized)
			return
		}
		if db == nil {
			http.Error(w, "la proyección del operador no está disponible", http.StatusServiceUnavailable)
			return
		}
		var overview operatorOverview
		overview.SchemaVersion = "1.0"
		overview.Source = "cholonet-network-read-model"
		overview.ReconciliationStatus = "UNKNOWN"
		overview.Incidents = []operatorIncident{}
		if err := db.QueryRowContext(r.Context(), `
			SELECT
				(SELECT COUNT(*) FROM payment_network),
				(SELECT COUNT(*) FROM payment_network WHERE status='ACTIVE'),
				(SELECT COUNT(*) FROM payment_network WHERE status='SUSPENDED'),
				(SELECT COUNT(*) FROM network_participant),
				(SELECT COUNT(*) FROM network_participant WHERE status='ACTIVE'),
				(SELECT COUNT(*) FROM network_participant WHERE status='SUSPENDED'),
				(SELECT COUNT(*) FROM directory_key WHERE status='ACTIVE'),
				(SELECT COUNT(*) FROM directory_key WHERE status='SUSPENDED')
		`).Scan(
			&overview.Health.NetworksTotal,
			&overview.Health.NetworksActive,
			&overview.Health.NetworksSuspended,
			&overview.Health.ParticipantsTotal,
			&overview.Health.ParticipantsActive,
			&overview.Health.ParticipantsSuspended,
			&overview.Health.ActiveKeys,
			&overview.Health.SuspendedKeys,
		); err != nil {
			http.Error(w, "la proyección del operador no está disponible", http.StatusServiceUnavailable)
			return
		}
		overview.Networks = overview.Health.NetworksTotal
		overview.Participants = overview.Health.ParticipantsTotal
		overview.ActiveKeys = overview.Health.ActiveKeys
		overview.SuspendedKeys = overview.Health.SuspendedKeys
		rows, err := db.QueryContext(r.Context(), `
			SELECT 'PARTICIPANT_SUSPENDED', COUNT(*), 'network_participant' FROM network_participant WHERE status='SUSPENDED' HAVING COUNT(*) > 0
			UNION ALL
			SELECT 'NETWORK_SUSPENDED', COUNT(*), 'payment_network' FROM payment_network WHERE status='SUSPENDED' HAVING COUNT(*) > 0
			UNION ALL
			SELECT 'DELIVERY_PENDING', COUNT(*), 'settlement_delivery' FROM settlement_delivery WHERE state='PENDING' HAVING COUNT(*) > 0
			UNION ALL
			SELECT code, COUNT(*), 'reconciliation_difference' FROM reconciliation_difference GROUP BY code
			UNION ALL
			SELECT 'TRANSFER_REJECTED', COUNT(*), 'transfer_instruction' FROM transfer_instruction WHERE state='REJECTED' HAVING COUNT(*) > 0
			ORDER BY 1
		`)
		if err != nil {
			http.Error(w, "la proyección del operador no está disponible", http.StatusServiceUnavailable)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var incident operatorIncident
			if err := rows.Scan(&incident.Code, &incident.Count, &incident.Source); err != nil {
				http.Error(w, "la proyección del operador no está disponible", http.StatusServiceUnavailable)
				return
			}
			overview.Incidents = append(overview.Incidents, incident)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, "la proyección del operador no está disponible", http.StatusServiceUnavailable)
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

func deliveryLeaseFromEnv(value string) (time.Duration, error) {
	if value == "" {
		value = "60"
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 5 || seconds > 3600 {
		return 0, fmt.Errorf("DELIVERY_LEASE_SECONDS debe estar entre 5 y 3600")
	}
	return time.Duration(seconds) * time.Second, nil
}

func main() {
	dbURL := os.Getenv("NETWORK_DB_URL")
	if dbURL == "" {
		slog.Error("se requiere NETWORK_DB_URL")
		os.Exit(1)
	}
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		slog.Error("abrir la base de datos de red", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		slog.Error("conectar con la base de datos de red", "error", err)
		os.Exit(1)
	}
	if err := instruction.Migrate(ctx, db); err != nil {
		slog.Error("migrar la base de datos de red", "error", err)
		os.Exit(1)
	}
	if err := settlement.Migrate(ctx, db); err != nil {
		slog.Error("migrar la base de datos de liquidación", "error", err)
		os.Exit(1)
	}
	routingStore := routing.NewStore(db)
	if err := routingStore.Migrate(ctx); err != nil {
		slog.Error("migrar la base de datos de enrutamiento", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	instruction.NewHandler(instruction.NewStore(db)).Register(mux)
	settlementStore := settlement.NewStore(db)
	settlement.NewHandler(settlementStore).Register(mux)
	if receiverURL := os.Getenv("BANK_RECEIVER_CREDIT_URL"); receiverURL != "" {
		lease, leaseErr := deliveryLeaseFromEnv(os.Getenv("DELIVERY_LEASE_SECONDS"))
		if leaseErr != nil {
			slog.Error("configuración inválida del lease de entrega", "error", leaseErr)
			os.Exit(1)
		}
		dispatcher := &settlement.DeliveryDispatcher{Store: settlementStore, ReceiverURL: receiverURL, Secret: os.Getenv("NETWORK_ADAPTER_SECRET"), Lease: lease}
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				_, _ = dispatcher.DispatchOnce(context.Background())
			}
		}()
	}
	settlement.NewReconciliationHandler(settlementStore).Register(mux)
	routing.NewHandler(routingStore).Register(mux)
	directoryStore := directory.NewStore(db)
	if err := directoryStore.Migrate(ctx); err != nil {
		slog.Error("migrar la base de datos de directorio", "error", err)
		os.Exit(1)
	}
	directory.NewHandler(directoryStore).Register(mux)
	messageStore := message.NewStore(db)
	if err := messageStore.Migrate(ctx); err != nil {
		slog.Error("migrar la base de datos de mensajes", "error", err)
		os.Exit(1)
	}
	message.NewHandler(messageStore).Register(mux)
	orchestration.NewHandler(orchestration.NewService(instruction.NewStore(db), routingStore, messageStore), os.Getenv("CHOLONET_HMAC_SECRET")).Register(mux)
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
	slog.Info("esqueleto de cholonet escuchando", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("el servidor se detuvo", "error", err)
		os.Exit(1)
	}
}
