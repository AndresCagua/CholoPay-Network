package routing

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/routes/resolve", h.resolve)
}

type operatorHandler struct {
	store *Store
	token string
}

func NewOperatorHandler(store *Store, token string) *operatorHandler {
	return &operatorHandler{store: store, token: token}
}

func (h *operatorHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/operator/networks", h.createNetwork)
	mux.HandleFunc("GET /internal/v1/operator/networks", h.listNetworks)
	mux.HandleFunc("GET /internal/v1/operator/networks/{network_id}", h.getNetwork)
	mux.HandleFunc("POST /internal/v1/operator/participants", h.createParticipant)
	mux.HandleFunc("GET /internal/v1/operator/participants", h.listParticipants)
	mux.HandleFunc("GET /internal/v1/operator/participants/{participant_id}", h.getParticipant)
	mux.HandleFunc("POST /internal/v1/operator/networks/{network_id}/suspend", h.suspendNetwork)
	mux.HandleFunc("POST /internal/v1/operator/networks/{network_id}/reactivate", h.reactivateNetwork)
	mux.HandleFunc("POST /internal/v1/operator/participants/{participant_id}/suspend", h.suspendParticipant)
	mux.HandleFunc("POST /internal/v1/operator/participants/{participant_id}/reactivate", h.reactivateParticipant)
}

type networkRequest struct {
	ID   uuid.UUID `json:"network_id"`
	Code string    `json:"code"`
	Kind string    `json:"kind"`
}

type participantRequest struct {
	ID        uuid.UUID `json:"participant_id"`
	NetworkID uuid.UUID `json:"network_id"`
	Code      string    `json:"code"`
}

type networkResponse struct {
	NetworkID uuid.UUID `json:"network_id"`
	Code      string    `json:"code"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
}

type networkDetailResponse struct {
	networkResponse
	Participants []participantSummary `json:"participants"`
}

type participantSummary struct {
	ParticipantID uuid.UUID `json:"participant_id"`
	NetworkID     uuid.UUID `json:"network_id"`
	Code          string    `json:"code"`
	Status        string    `json:"status"`
}

func (h *operatorHandler) authorized(r *http.Request) bool {
	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, "Bearer ") || h.token == "" {
		return false
	}
	provided := strings.TrimPrefix(value, "Bearer ")
	return len(provided) == len(h.token) && subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) == 1
}

func (h *operatorHandler) guard(w http.ResponseWriter, r *http.Request) bool {
	if h.authorized(r) {
		return true
	}
	http.Error(w, "se requiere autorización de operador", http.StatusUnauthorized)
	return false
}

func (h *operatorHandler) listNetworks(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	networks, err := h.store.ListNetworks(r.Context())
	if err != nil {
		http.Error(w, "las redes no están disponibles", http.StatusServiceUnavailable)
		return
	}
	response := make([]networkResponse, 0, len(networks))
	for _, network := range networks {
		response = append(response, networkResponse{NetworkID: network.ID, Code: network.Code, Kind: network.Kind, Status: network.Status})
	}
	writeJSON(w, http.StatusOK, map[string]any{"networks": response})
}

func (h *operatorHandler) getNetwork(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("network_id"))
	if err != nil {
		http.Error(w, "network_id inválido", http.StatusBadRequest)
		return
	}
	detail, err := h.store.GetNetwork(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "la red no existe", http.StatusNotFound)
			return
		}
		http.Error(w, "la red no está disponible", http.StatusServiceUnavailable)
		return
	}
	participants := make([]participantSummary, 0, len(detail.Participants))
	for _, participant := range detail.Participants {
		participants = append(participants, participantSummary{ParticipantID: participant.ID, NetworkID: participant.NetworkID, Code: participant.Code, Status: participant.Status})
	}
	writeJSON(w, http.StatusOK, networkDetailResponse{
		networkResponse: networkResponse{NetworkID: detail.ID, Code: detail.Code, Kind: detail.Kind, Status: detail.Status},
		Participants:    participants,
	})
}

func (h *operatorHandler) createNetwork(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var request networkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "solicitud inválida", http.StatusBadRequest)
		return
	}
	if request.ID == uuid.Nil {
		request.ID = uuid.New()
	}
	network := Network{ID: request.ID, Code: request.Code, Kind: request.Kind, Status: Active}
	if err := h.store.RegisterNetwork(r.Context(), network); err != nil {
		if errors.Is(err, ErrInvalidNetwork) {
			http.Error(w, "red inválida", http.StatusBadRequest)
			return
		}
		http.Error(w, "la red ya existe o no pudo registrarse", http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusCreated, network)
}

func (h *operatorHandler) createParticipant(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var request participantRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "solicitud inválida", http.StatusBadRequest)
		return
	}
	if request.ID == uuid.Nil {
		request.ID = uuid.New()
	}
	participant := Participant{ID: request.ID, NetworkID: request.NetworkID, Code: request.Code, Status: Active}
	if err := h.store.RegisterParticipant(r.Context(), participant); err != nil {
		if errors.Is(err, ErrInvalidParticipant) {
			http.Error(w, "participante inválido", http.StatusBadRequest)
			return
		}
		http.Error(w, "el participante ya existe o la red no existe", http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusCreated, participant)
}

type participantResponse struct {
	ParticipantID     uuid.UUID `json:"participant_id"`
	NetworkID         uuid.UUID `json:"network_id"`
	ParticipantCode   string    `json:"participant_code"`
	ParticipantStatus string    `json:"participant_status"`
	NetworkCode       string    `json:"network_code"`
	NetworkKind       string    `json:"network_kind"`
	NetworkStatus     string    `json:"network_status"`
}

func (h *operatorHandler) listParticipants(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	participants, err := h.store.ListParticipants(r.Context())
	if err != nil {
		http.Error(w, "los participantes no están disponibles", http.StatusServiceUnavailable)
		return
	}
	response := make([]participantResponse, 0, len(participants))
	for _, detail := range participants {
		response = append(response, participantResponse{
			ParticipantID: detail.ID, NetworkID: detail.NetworkID,
			ParticipantCode: detail.Code, ParticipantStatus: detail.Status,
			NetworkCode: detail.NetworkCode, NetworkKind: detail.NetworkKind,
			NetworkStatus: detail.NetworkStatus,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"participants": response})
}

func (h *operatorHandler) getParticipant(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("participant_id"))
	if err != nil {
		http.Error(w, "participant_id inválido", http.StatusBadRequest)
		return
	}
	detail, err := h.store.GetParticipant(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "el participante no existe", http.StatusNotFound)
			return
		}
		http.Error(w, "el participante no está disponible", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, participantResponse{
		ParticipantID: detail.ID, NetworkID: detail.NetworkID,
		ParticipantCode: detail.Code, ParticipantStatus: detail.Status,
		NetworkCode: detail.NetworkCode, NetworkKind: detail.NetworkKind,
		NetworkStatus: detail.NetworkStatus,
	})
}

func (h *operatorHandler) suspendNetwork(w http.ResponseWriter, r *http.Request) {
	h.setNetworkStatus(w, r, Suspended)
}
func (h *operatorHandler) reactivateNetwork(w http.ResponseWriter, r *http.Request) {
	h.setNetworkStatus(w, r, Active)
}
func (h *operatorHandler) suspendParticipant(w http.ResponseWriter, r *http.Request) {
	h.setParticipantStatus(w, r, Suspended)
}
func (h *operatorHandler) reactivateParticipant(w http.ResponseWriter, r *http.Request) {
	h.setParticipantStatus(w, r, Active)
}

func (h *operatorHandler) setNetworkStatus(w http.ResponseWriter, r *http.Request, status string) {
	if !h.guard(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("network_id"))
	if err != nil {
		http.Error(w, "network_id inválido", http.StatusBadRequest)
		return
	}
	if err := h.store.SetNetworkStatus(r.Context(), id, status); err != nil {
		writeStatusError(w, err, "la red")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"network_id": id, "status": status})
}

func (h *operatorHandler) setParticipantStatus(w http.ResponseWriter, r *http.Request, status string) {
	if !h.guard(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("participant_id"))
	if err != nil {
		http.Error(w, "participant_id inválido", http.StatusBadRequest)
		return
	}
	if err := h.store.SetParticipantStatus(r.Context(), id, status); err != nil {
		writeStatusError(w, err, "el participante")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participant_id": id, "status": status})
}

func writeStatusError(w http.ResponseWriter, err error, resource string) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, resource+" no existe", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrInvalidNetwork) || errors.Is(err, ErrInvalidParticipant) {
		http.Error(w, "identificador inválido", http.StatusBadRequest)
		return
	}
	http.Error(w, resource+" no pudo actualizarse", http.StatusConflict)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
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
