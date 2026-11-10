package routing

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/routes/resolve", h.resolve)
}

type routeRequest struct {
	SourceParticipantID      uuid.UUID `json:"source_participant_id"`
	DestinationParticipantID uuid.UUID `json:"destination_participant_id"`
}
type routeResponse struct {
	RouteKind            RouteKind `json:"route_kind"`
	SourceNetworkID      uuid.UUID `json:"source_network_id"`
	DestinationNetworkID uuid.UUID `json:"destination_network_id"`
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	var request routeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "solicitud inválida", http.StatusBadRequest)
		return
	}
	route, err := h.store.Route(r.Context(), request.SourceParticipantID, request.DestinationParticipantID)
	if err != nil {
		if errors.Is(err, ErrParticipantUnavailable) {
			http.Error(w, "participante no disponible", http.StatusConflict)
		} else {
			http.Error(w, "ruta no disponible", http.StatusNotFound)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(routeResponse{RouteKind: route.Kind, SourceNetworkID: route.SourceNetwork.ID, DestinationNetworkID: route.DestinationNetwork.ID})
}
