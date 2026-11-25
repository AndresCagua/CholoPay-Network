package settlement

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
)

type ReconciliationHandler struct{ store *Store }

func NewReconciliationHandler(store *Store) *ReconciliationHandler {
	return &ReconciliationHandler{store: store}
}
func (h *ReconciliationHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/reconciliation/runs", h.create)
	mux.HandleFunc("GET /internal/v1/reconciliation/runs/{id}", h.get)
}
func (h *ReconciliationHandler) create(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	var targetTransferID *uuid.UUID
	if raw := r.URL.Query().Get("transfer_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			http.Error(w, "transfer_id inválido", http.StatusBadRequest)
			return
		}
		targetTransferID = &parsed
	}
	result, err := h.store.ReconcileForTransfer(r.Context(), uuid.New(), targetTransferID)
	if err != nil {
		http.Error(w, "falló la conciliación", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
func (h *ReconciliationHandler) get(w http.ResponseWriter, r *http.Request) {
	if !authorizedReplayBearer(r) {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identificador de ejecución inválido", http.StatusBadRequest)
		return
	}
	result, err := h.store.ReconciliationRun(r.Context(), id)
	if err != nil {
		http.Error(w, "ejecución de conciliación no encontrada", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
