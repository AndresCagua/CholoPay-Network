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
	result, err := h.store.Reconcile(r.Context(), uuid.New())
	if err != nil {
		http.Error(w, "reconciliation failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
func (h *ReconciliationHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}
	result, err := h.store.ReconciliationRun(r.Context(), id)
	if err != nil {
		http.Error(w, "reconciliation run not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
