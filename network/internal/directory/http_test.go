package directory

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
