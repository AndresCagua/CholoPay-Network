package settlement

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseReceiverRoutesBuildsIndependentTargets(t *testing.T) {
	costa := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	sierra := uuid.MustParse("00000000-0000-4000-8000-000000000102")
	routes, err := ParseReceiverRoutes("00000000-0000-4000-8000-000000000101=http://bank-core/internal/v1/receiver-credits|costa-secret,00000000-0000-4000-8000-000000000102=http://bank-core-sierra/internal/v1/receiver-credits|sierra-secret")
	if err != nil {
		t.Fatal(err)
	}
	if routes[costa].URL != "http://bank-core/internal/v1/receiver-credits" || routes[costa].Secret != "costa-secret" {
		t.Fatalf("ruta de Costa inesperada: %+v", routes[costa])
	}
	if routes[sierra].URL != "http://bank-core-sierra/internal/v1/receiver-credits" || routes[sierra].Secret != "sierra-secret" {
		t.Fatalf("ruta de Sierra inesperada: %+v", routes[sierra])
	}
}

func TestParseReceiverRoutesRejectsDuplicateOrIncompleteTargets(t *testing.T) {
	cases := []string{
		"not-a-route",
		"00000000-0000-4000-8000-000000000101=http://bank-core|",
		"00000000-0000-4000-8000-000000000101=http://one|secret,00000000-0000-4000-8000-000000000101=http://two|secret",
	}
	for _, value := range cases {
		if _, err := ParseReceiverRoutes(value); err == nil {
			t.Fatalf("se esperaba rechazar la ruta %q", value)
		}
	}
}

func TestReceiverTargetUsesParticipantRouteAndRejectsUnknownParticipant(t *testing.T) {
	participant := uuid.MustParse("00000000-0000-4000-8000-000000000102")
	dispatcher := &DeliveryDispatcher{ReceiverTargets: map[uuid.UUID]ReceiverTarget{
		participant: {URL: "http://bank-core-sierra/internal/v1/receiver-credits", Secret: "sierra-secret"},
	}}
	target, err := dispatcher.receiverTarget(participant)
	if err != nil || target.URL != "http://bank-core-sierra/internal/v1/receiver-credits" || target.Secret != "sierra-secret" {
		t.Fatalf("ruta de Sierra inesperada: %+v/%v", target, err)
	}
	if _, err := dispatcher.receiverTarget(uuid.New()); err == nil {
		t.Fatal("se esperaba rechazar un participante sin ruta")
	}
}

func TestReceiverTargetKeepsLegacySingleReceiverFallback(t *testing.T) {
	dispatcher := &DeliveryDispatcher{ReceiverURL: "http://bank-core/internal/v1/receiver-credits", Secret: "legacy-secret"}
	target, err := dispatcher.receiverTarget(uuid.New())
	if err != nil || target.URL != dispatcher.ReceiverURL || target.Secret != dispatcher.Secret {
		t.Fatalf("fallback legacy inesperado: %+v/%v", target, err)
	}
}
