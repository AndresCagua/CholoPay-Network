package main

import (
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
