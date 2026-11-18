package settlement

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ReceiverTarget identifica el endpoint y la credencial sintética de un banco receptor.
type ReceiverTarget struct {
	URL    string
	Secret string
}

// ParseReceiverRoutes interpreta: participant-uuid=url|secret,participant-uuid=url|secret.
func ParseReceiverRoutes(value string) (map[uuid.UUID]ReceiverTarget, error) {
	routes := make(map[uuid.UUID]ReceiverTarget)
	if strings.TrimSpace(value) == "" {
		return routes, nil
	}
	for _, rawRoute := range strings.Split(value, ",") {
		parts := strings.SplitN(strings.TrimSpace(rawRoute), "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("BANK_RECEIVER_ROUTES contiene una ruta sin separador")
		}
		participantID, err := uuid.Parse(strings.TrimSpace(parts[0]))
		if err != nil || participantID == uuid.Nil {
			return nil, fmt.Errorf("BANK_RECEIVER_ROUTES contiene un participante inválido: %q", parts[0])
		}
		endpoint := strings.SplitN(parts[1], "|", 2)
		if len(endpoint) != 2 || strings.TrimSpace(endpoint[0]) == "" || strings.TrimSpace(endpoint[1]) == "" {
			return nil, errors.New("BANK_RECEIVER_ROUTES requiere URL y secreto para cada participante")
		}
		if _, exists := routes[participantID]; exists {
			return nil, fmt.Errorf("BANK_RECEIVER_ROUTES duplica el participante %s", participantID)
		}
		routes[participantID] = ReceiverTarget{URL: strings.TrimSpace(endpoint[0]), Secret: strings.TrimSpace(endpoint[1])}
	}
	return routes, nil
}
