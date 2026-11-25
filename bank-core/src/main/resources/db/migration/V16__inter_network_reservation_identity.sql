ALTER TABLE balance_reservation ADD COLUMN IF NOT EXISTS request_hash CHAR(64);
ALTER TABLE balance_reservation ADD COLUMN IF NOT EXISTS currency CHAR(3) NOT NULL DEFAULT 'USD';
