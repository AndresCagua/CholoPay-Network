package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperatorAuthorizationRequiresStrictBearerScheme(t *testing.T) {
	handler := &operatorHandler{token: "expected-token"}
	for _, header := range []string{"expected-token", "Basic expected-token", "Bearer", "Bearer ", "Bearer expected-token ", " Bearer expected-token", "Bearer expected-token extra"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		if handler.authorized(req) {
			t.Fatalf("se aceptó un header inválido %q", header)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	if !handler.authorized(req) {
		t.Fatal("se rechazó el bearer válido")
	}
}
