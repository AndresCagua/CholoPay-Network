-- Fixtures sintéticos de recepción específicos del participante bancario.
DO $$
BEGIN
  IF '${participant_id}' = 'BANCO_SIERRA' THEN
    UPDATE ledger_account
    SET posted_balance_minor = 1000000
    WHERE id = '00000000-0000-4000-8000-000000000012';
    INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
    VALUES ('00000000-0000-4000-8000-000000000013', 'synthetic-receiver-sierra', 'USD', 0)
    ON CONFLICT (id) DO NOTHING;
  ELSIF '${participant_id}' = 'BANCO_ORIENTE' THEN
    UPDATE ledger_account
    SET posted_balance_minor = 1000000
    WHERE id = '00000000-0000-4000-8000-000000000022';
    INSERT INTO ledger_account (id, owner_ref, currency, posted_balance_minor)
    VALUES ('00000000-0000-4000-8000-000000000023', 'synthetic-receiver-oriente', 'USD', 0)
    ON CONFLICT (id) DO NOTHING;
  END IF;
END $$;
