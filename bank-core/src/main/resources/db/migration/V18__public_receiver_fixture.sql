-- Identidad pública sintética para demostrar la frontera HTTP del receptor Sierra.
DO $$
BEGIN
  IF '${participant_id}' = 'BANCO_SIERRA' THEN
    INSERT INTO public_customer(id, participant_id, synthetic_reference, api_token_hash)
    VALUES (
      '00000000-0000-4000-8000-000000000031',
      'BANCO_SIERRA',
      'phase20-public-receiver',
      '8679575ed0244b42c6a2c0878659c18cb38dcb326e677c962c7d8e0fba67ea03'
    )
    ON CONFLICT (id) DO NOTHING;

    UPDATE ledger_account
       SET participant_id = 'BANCO_SIERRA',
           customer_id = '00000000-0000-4000-8000-000000000031'
     WHERE id = '00000000-0000-4000-8000-000000000013';
  END IF;
END $$;
