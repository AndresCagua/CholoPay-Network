package settlement

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"
	"time"
)

type ReceiverCreditPayload struct {
	DeliveryID           uuid.UUID `json:"deliveryId"`
	EventID              uuid.UUID `json:"eventId"`
	TransferID           uuid.UUID `json:"transferId"`
	DestinationAccountID uuid.UUID `json:"destinationAccountId"`
	ClearingAccountID    uuid.UUID `json:"clearingAccountId"`
	AmountMinor          int64     `json:"amountMinor"`
}

func (p ReceiverCreditPayload) Canonical() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s:%d:USD", p.DeliveryID, p.EventID, p.TransferID, p.DestinationAccountID, p.ClearingAccountID, p.AmountMinor)
}
func SignedReceiverRequest(method, url, secret string, p ReceiverCreditPayload, now time.Time) (*http.Request, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	ts := strconv.FormatInt(now.Unix(), 10)
	requestID := p.DeliveryID.String()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + requestID + "." + p.Canonical()))
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CholoPay-Request-Timestamp", ts)
	req.Header.Set("X-CholoPay-Request-Id", requestID)
	req.Header.Set("X-CholoPay-Signature", hex.EncodeToString(mac.Sum(nil)))
	return req, nil
}

type DeliveryDispatcher struct {
	Store       *Store
	Client      *http.Client
	ReceiverURL string
	Secret      string
	Lease       time.Duration
	MaxAttempts int
	Backoff     func(int) time.Duration
	Now         func() time.Time
}

func (d *DeliveryDispatcher) DispatchOnce(ctx context.Context) (bool, error) {
	if d.Client == nil {
		d.Client = http.DefaultClient
	}
	if d.Lease == 0 {
		d.Lease = time.Minute
	}
	if d.MaxAttempts == 0 {
		d.MaxAttempts = 5
	}
	if d.Backoff == nil {
		d.Backoff = func(attempt int) time.Duration {
			if attempt > 6 {
				attempt = 6
			}
			return time.Duration(1<<uint(attempt-1)) * time.Second
		}
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if err := d.Store.EnqueueEligibleDeliveries(ctx); err != nil {
		return false, err
	}
	owner := uuid.New()
	delivery, err := d.Store.ClaimDelivery(ctx, owner, d.Lease)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	p := ReceiverCreditPayload{delivery.DeliveryID, delivery.EventID, delivery.TransferID, delivery.DestinationAccountID, delivery.ClearingAccountID, delivery.AmountMinor}
	req, err := SignedReceiverRequest(http.MethodPost, d.ReceiverURL, d.Secret, p, d.Now())
	if err != nil {
		return true, d.recordFailure(ctx, delivery, owner, "CONTRACT", err.Error())
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return true, d.recordRetry(ctx, delivery, owner, err.Error())
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, err = d.Store.AcknowledgeDelivery(ctx, delivery.DeliveryID, delivery.ReceiverParticipantID, delivery.EventID)
		return true, err
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 400 || resp.StatusCode == 409 {
		return true, d.recordFailure(ctx, delivery, owner, "RECEIVER_"+strconv.Itoa(resp.StatusCode), "receiver rejected delivery")
	}
	return true, d.recordRetry(ctx, delivery, owner, "ambiguous receiver transport")
}
func (d *DeliveryDispatcher) recordRetry(ctx context.Context, delivery Delivery, owner uuid.UUID, detail string) error {
	if delivery.AttemptCount >= d.MaxAttempts {
		return d.recordFailure(ctx, delivery, owner, "DELIVERY_DLQ", detail)
	}
	_, err := d.Store.db.ExecContext(ctx, `UPDATE settlement_delivery SET next_attempt_at=CURRENT_TIMESTAMP+$1::interval,lease_owner=NULL,lease_until=NULL,updated_at=CURRENT_TIMESTAMP,failure_detail=$2 WHERE delivery_id=$3 AND lease_owner=$4`, d.Backoff(delivery.AttemptCount).String(), detail, delivery.DeliveryID, owner)
	return err
}
func (d *DeliveryDispatcher) recordFailure(ctx context.Context, delivery Delivery, owner uuid.UUID, code, detail string) error {
	_, err := d.Store.db.ExecContext(ctx, `UPDATE settlement_delivery SET state='FAILED',failure_code=$1,failure_detail=$2,lease_owner=NULL,lease_until=NULL,updated_at=CURRENT_TIMESTAMP WHERE delivery_id=$3 AND lease_owner=$4`, code, detail, delivery.DeliveryID, owner)
	return err
}

var ErrNoDelivery = errors.New("no hay entrega pendiente")
