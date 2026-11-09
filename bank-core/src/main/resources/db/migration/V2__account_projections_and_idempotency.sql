ALTER TABLE ledger_account
  ADD COLUMN posted_balance_minor BIGINT NOT NULL DEFAULT 0 CHECK (posted_balance_minor >= 0),
  ADD COLUMN reserved_balance_minor BIGINT NOT NULL DEFAULT 0 CHECK (reserved_balance_minor >= 0),
  ADD CONSTRAINT available_balance_non_negative CHECK (posted_balance_minor >= reserved_balance_minor);

ALTER TABLE journal ADD COLUMN request_hash CHAR(64) NOT NULL DEFAULT repeat('0', 64);
CREATE UNIQUE INDEX journal_idempotency_request ON journal(idempotency_key, request_hash);
