CREATE INDEX public_movement_account_created_id
  ON journal_entry(account_id, created_at DESC, id DESC);

CREATE INDEX public_journal_entry_journal
  ON journal_entry(journal_id, id);
