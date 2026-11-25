package directory

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestDirectoryAuthorizationRequiresStrictBearerScheme(t *testing.T) {
	t.Setenv("DIRECTORY_PORT_TOKEN", "expected-token")
	for _, header := range []string{"expected-token", "Basic expected-token", "Bearer", "Bearer ", "Bearer expected-token ", " Bearer expected-token", "Bearer expected-token extra"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		if authorizedBearer(req, "expected-token") {
			t.Fatalf("se aceptó un header inválido %q", header)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	if !authorizedBearer(req, "expected-token") {
		t.Fatal("se rechazó el bearer válido")
	}
}

func TestParticipantAuthorizationRequiresStrictBearerScheme(t *testing.T) {
	participantID := uuid.New()
	t.Setenv("DIRECTORY_PARTICIPANT_TOKENS", participantID.String()+"=expected-token")
	for _, header := range []string{"expected-token", "Basic expected-token", "Bearer", "Bearer ", "Bearer expected-token ", " Bearer expected-token", "Bearer expected-token extra"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		if authorizedParticipant(req, participantID) {
			t.Fatalf("se aceptó un header de participante inválido %q", header)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	if !authorizedParticipant(req, participantID) {
		t.Fatal("se rechazó el bearer de participante válido")
	}
}

func TestDirectoryLifecycleEndpointsRequireInternalAuthorization(t *testing.T) {
	t.Setenv("DIRECTORY_PORT_TOKEN", "expected-token")
	mux := http.NewServeMux()
	NewHandler(nil).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/directory/keys", bytes.NewReader([]byte(`{}`)))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba autorización del directorio, se obtuvo %d", response.Code)
	}
}
