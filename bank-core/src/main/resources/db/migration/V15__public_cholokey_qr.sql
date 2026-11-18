CREATE TABLE public_cholokey_qr_dynamic (
  payload_id UUID PRIMARY KEY,
  key_id UUID NOT NULL,
  payload_hash CHAR(64) NOT NULL UNIQUE,
  amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
  reference VARCHAR(160) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX public_cholokey_qr_dynamic_expiry ON public_cholokey_qr_dynamic(expires_at);
