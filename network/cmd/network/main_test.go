package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperatorOverviewRequiresAuthorization(t *testing.T) {
	t.Setenv("OPERATOR_CONSOLE_TOKEN", "expected-token")
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/operator/overview", nil)
	response := httptest.NewRecorder()
	operatorOverviewHandler(nil)(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized operator request, got %d", response.Code)
	}
}

func TestOperatorOverviewDoesNotInventProjectionWhenNetworkSourceUnavailable(t *testing.T) {
	t.Setenv("OPERATOR_CONSOLE_TOKEN", "expected-token")
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/operator/overview", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	response := httptest.NewRecorder()
	operatorOverviewHandler(nil)(response, req)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected unavailable projection, got %d", response.Code)
	}
}

func TestOperatorOverviewProjectionContractIsAggregated(t *testing.T) {
	payload, err := json.Marshal(operatorOverview{
		SchemaVersion:        "1.0",
		Source:               "cholonet-network-read-model",
		LastSynchronizedAt:   "2026-01-01T00:00:00Z",
		ReconciliationStatus: "UNKNOWN",
		Networks:             1,
		Participants:         2,
		ActiveKeys:           1,
		SuspendedKeys:        1,
		Health:               operatorHealth{ParticipantsTotal: 2, ParticipantsActive: 1, ParticipantsSuspended: 1},
		Incidents:            []operatorIncident{{Code: "DELIVERY_PENDING", Count: 1, Source: "settlement_delivery"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["participant_id"]; ok {
		t.Fatal("operator projection must not expose participant identifiers")
	}
	if decoded["source"] != "cholonet-network-read-model" || decoded["reconciliation_status"] != "UNKNOWN" {
		t.Fatalf("missing projection metadata: %s", payload)
	}
	if decoded["participants"] != float64(2) || decoded["active_keys"] != float64(1) {
		t.Fatalf("legacy aggregate fields changed: %s", payload)
	}
}
