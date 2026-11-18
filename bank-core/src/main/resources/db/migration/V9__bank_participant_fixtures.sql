-- Fixtures sintéticos separados para los participantes Sierra y Oriente.
INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
VALUES
  ('00000000-0000-4000-8000-000000000011', 'synthetic-electricity-source-sierra', 'USD', 1000000),
  ('00000000-0000-4000-8000-000000000012', 'synthetic-utilities-clearing-sierra', 'USD', 0),
  ('00000000-0000-4000-8000-000000000021', 'synthetic-electricity-source-oriente', 'USD', 1000000),
  ('00000000-0000-4000-8000-000000000022', 'synthetic-utilities-clearing-oriente', 'USD', 0)
ON CONFLICT (id) DO NOTHING;
