-- Fixtures sintéticos para acreditar entregas recibidas por Sierra y Oriente.
UPDATE ledger_account
SET posted_balance_minor = 1000000
WHERE id IN (
  '00000000-0000-4000-8000-000000000012',
  '00000000-0000-4000-8000-000000000022'
);

INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
VALUES
  ('00000000-0000-4000-8000-000000000013', 'synthetic-receiver-sierra', 'USD', 0),
  ('00000000-0000-4000-8000-000000000023', 'synthetic-receiver-oriente', 'USD', 0)
ON CONFLICT (id) DO NOTHING;
