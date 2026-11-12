package settlement

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDefaultReceiverHTTPClientHasBoundedTimeout(t *testing.T) {
	if got := defaultReceiverHTTPClient().Timeout; got != 30*time.Second {
		t.Fatalf("se esperaba timeout HTTP de 30 segundos, se obtuvo %s", got)
	}
}

func TestDurationSecondsRejectsTextualOrFractionalIntervals(t *testing.T) {
	if got, err := durationSeconds(time.Minute); err != nil || got != 60 {
		t.Fatalf("se esperaban 60 segundos, se obtuvo %d/%v", got, err)
	}
	for _, value := range []time.Duration{0, -time.Second, 1500 * time.Millisecond} {
		if _, err := durationSeconds(value); err == nil {
			t.Fatalf("se esperaba rechazar la duración %s", value)
		}
	}
}

func TestSignedReceiverRequestUsesStableDeliveryIdentityAndCanonicalPayload(t *testing.T) {
	p := ReceiverCreditPayload{DeliveryID: uuid.New(), EventID: uuid.New(), TransferID: uuid.New(), DestinationAccountID: uuid.New(), ClearingAccountID: uuid.New(), AmountMinor: 125}
	now := time.Unix(1700000000, 0)
	req, err := SignedReceiverRequest(http.MethodPost, "http://bank-core/internal/v1/receiver-credits", "secret", p, now)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("X-CholoPay-Request-Id") != p.DeliveryID.String() {
		t.Fatalf("request id changed: %s", req.Header.Get("X-CholoPay-Request-Id"))
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("1700000000." + p.DeliveryID.String() + "." + p.Canonical()))
	if req.Header.Get("X-CholoPay-Signature") != hex.EncodeToString(mac.Sum(nil)) {
		t.Fatal("signature does not cover timestamp, request id, and canonical payload")
	}
	if len(body) == 0 {
		t.Fatal("empty receiver payload")
	}
}
