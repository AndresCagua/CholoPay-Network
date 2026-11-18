-- Elimina fixtures históricos de otros participantes y deja solo los del banco actual.
DO $$
BEGIN
  IF '${participant_id}' = 'BANCO_SIERRA' THEN
    DELETE FROM ledger_account
    WHERE id IN (
      '00000000-0000-4000-8000-000000000021',
      '00000000-0000-4000-8000-000000000022',
      '00000000-0000-4000-8000-000000000023'
    );
  ELSIF '${participant_id}' = 'BANCO_ORIENTE' THEN
    DELETE FROM ledger_account
    WHERE id IN (
      '00000000-0000-4000-8000-000000000011',
      '00000000-0000-4000-8000-000000000012',
      '00000000-0000-4000-8000-000000000013'
    );
  ELSE
    DELETE FROM ledger_account
    WHERE id IN (
      '00000000-0000-4000-8000-000000000011',
      '00000000-0000-4000-8000-000000000012',
      '00000000-0000-4000-8000-000000000013',
      '00000000-0000-4000-8000-000000000021',
      '00000000-0000-4000-8000-000000000022',
      '00000000-0000-4000-8000-000000000023'
    );
  END IF;
END $$;

DO $$
BEGIN
  IF '${participant_id}' = 'BANCO_SIERRA' THEN
    INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
    VALUES
      ('00000000-0000-4000-8000-000000000011', 'synthetic-electricity-source-sierra', 'USD', 1000000),
      ('00000000-0000-4000-8000-000000000012', 'synthetic-utilities-clearing-sierra', 'USD', 1000000),
      ('00000000-0000-4000-8000-000000000013', 'synthetic-receiver-sierra', 'USD', 0)
    ON CONFLICT (id) DO NOTHING;
  ELSIF '${participant_id}' = 'BANCO_ORIENTE' THEN
    INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
    VALUES
      ('00000000-0000-4000-8000-000000000021', 'synthetic-electricity-source-oriente', 'USD', 1000000),
      ('00000000-0000-4000-8000-000000000022', 'synthetic-utilities-clearing-oriente', 'USD', 1000000),
      ('00000000-0000-4000-8000-000000000023', 'synthetic-receiver-oriente', 'USD', 0)
    ON CONFLICT (id) DO NOTHING;
  END IF;
END $$;
