package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestOperatorOverviewReadsAggregatesWithoutExposingSecretsOrMutatingSources(t *testing.T) {
	t.Setenv("OPERATOR_CONSOLE_TOKEN", "expected-token")
	state := &operatorTestDBState{}
	db := openOperatorTestDB(t, state)
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/operator/overview", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	response := httptest.NewRecorder()
	operatorOverviewHandler(db)(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected operator projection, got %d: %s", response.Code, response.Body)
	}
	body := response.Body.String()
	for _, forbidden := range []string{"participant_id", "secret", "token", "account_number", "wallet_address"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("operator projection exposed forbidden field %q: %s", forbidden, body)
		}
	}
	var overview operatorOverview
	if err := json.Unmarshal(response.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if overview.Participants != 3 || overview.ActiveKeys != 2 || overview.SuspendedKeys != 1 {
		t.Fatalf("expected aggregate counts, got %+v", overview)
	}
	if len(overview.Incidents) != 1 || overview.Incidents[0].Count != 2 {
		t.Fatalf("expected aggregate incident count, got %+v", overview.Incidents)
	}
	if len(state.queries) != 2 || state.execs != 0 {
		t.Fatalf("operator projection must be read-only: queries=%v execs=%d", state.queries, state.execs)
	}
	for _, query := range state.queries {
		if !strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
			t.Fatalf("operator projection issued non-read query: %s", query)
		}
	}
}

func TestOperatorOverviewReturnsUnavailableWhenSourceQueryFails(t *testing.T) {
	t.Setenv("OPERATOR_CONSOLE_TOKEN", "expected-token")
	state := &operatorTestDBState{queryErr: errors.New("source unavailable")}
	db := openOperatorTestDB(t, state)
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/operator/overview", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	response := httptest.NewRecorder()
	operatorOverviewHandler(db)(response, req)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected unavailable projection, got %d: %s", response.Code, response.Body)
	}
	if strings.Contains(response.Body.String(), "0") || strings.Contains(response.Body.String(), "stale") {
		t.Fatalf("source failure must not return invented or stale projection: %s", response.Body)
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

type operatorTestDBState struct {
	mu       sync.Mutex
	queries  []string
	execs    int
	queryErr error
}

type operatorTestDriver struct {
	state *operatorTestDBState
}

type operatorTestConn struct {
	state *operatorTestDBState
}

type operatorTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

var registerOperatorTestDriver sync.Once

func openOperatorTestDB(t *testing.T, state *operatorTestDBState) *sql.DB {
	t.Helper()
	registerOperatorTestDriver.Do(func() {
		sql.Register("operator-test", operatorTestDriver{})
	})
	operatorTestDriverRegistry.mu.Lock()
	operatorTestDriverRegistry.state = state
	operatorTestDriverRegistry.mu.Unlock()
	db, err := sql.Open("operator-test", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

var operatorTestDriverRegistry struct {
	mu    sync.Mutex
	state *operatorTestDBState
}

func (operatorTestDriver) Open(string) (driver.Conn, error) {
	operatorTestDriverRegistry.mu.Lock()
	state := operatorTestDriverRegistry.state
	operatorTestDriverRegistry.mu.Unlock()
	return &operatorTestConn{state: state}, nil
}

func (c *operatorTestConn) Close() error { return nil }
func (c *operatorTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c *operatorTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not supported")
}

func (c *operatorTestConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	c.state.queries = append(c.state.queries, query)
	err := c.state.queryErr
	c.state.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if strings.Contains(query, "COUNT(*) FROM payment_network") {
		return &operatorTestRows{
			columns: []string{"networks_total", "networks_active", "networks_suspended", "participants_total", "participants_active", "participants_suspended", "active_keys", "suspended_keys"},
			values:  [][]driver.Value{{int64(1), int64(1), int64(0), int64(3), int64(2), int64(1), int64(2), int64(1)}},
		}, nil
	}
	return &operatorTestRows{
		columns: []string{"code", "count", "source"},
		values:  [][]driver.Value{{"PARTICIPANT_SUSPENDED", int64(2), "network_participant"}},
	}, nil
}

func (c *operatorTestConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	c.state.execs++
	c.state.mu.Unlock()
	return nil, errors.New("mutation attempted")
}

func (r *operatorTestRows) Columns() []string { return r.columns }
func (r *operatorTestRows) Close() error      { return nil }
func (r *operatorTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}
