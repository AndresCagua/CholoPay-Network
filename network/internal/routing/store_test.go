package routing

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresRouteSeparatesNetworkDomains(t *testing.T) {
	url := os.Getenv("NETWORK_DB_URL")
	if url == "" {
		t.Skip("NETWORK_DB_URL es necesario para la prueba de integración de Docker")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE network_participant, payment_network CASCADE"); err != nil {
		t.Fatal(err)
	}
	banking := Network{ID: uuid.New(), Code: "CHOLOPAY_BANKING", Kind: "BANKING", Status: "ACTIVE"}
	fintech := Network{ID: uuid.New(), Code: "CHOLOPAY_FINTECH", Kind: "FINTECH", Status: "ACTIVE"}
	if err := store.RegisterNetwork(ctx, banking); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterNetwork(ctx, fintech); err != nil {
		t.Fatal(err)
	}
	costa := Participant{ID: uuid.New(), NetworkID: banking.ID, Code: "COSTA", Status: "ACTIVE"}
	oriente := Participant{ID: uuid.New(), NetworkID: banking.ID, Code: "ORIENTE", Status: "ACTIVE"}
	walletA := Participant{ID: uuid.New(), NetworkID: fintech.ID, Code: "WALLET_A", Status: "ACTIVE"}
	for _, p := range []Participant{costa, oriente, walletA} {
		if err := store.RegisterParticipant(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	intra, err := store.Route(ctx, costa.ID, oriente.ID)
	if err != nil {
		t.Fatal(err)
	}
	if intra.Kind != IntraNetwork {
		t.Fatalf("se esperaba una ruta intrared, se obtuvo %s", intra.Kind)
	}
	inter, err := store.Route(ctx, costa.ID, walletA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if inter.Kind != InterNetwork || inter.DestinationNetwork.ID != fintech.ID {
		t.Fatalf("ruta interred inesperada: %+v", inter)
	}
	if _, err := db.ExecContext(ctx, "UPDATE network_participant SET status='SUSPENDED' WHERE participant_id=$1", walletA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Route(ctx, costa.ID, walletA.ID); err != ErrParticipantUnavailable {
		t.Fatalf("se esperaba un participante no disponible, se obtuvo %v", err)
	}
}
