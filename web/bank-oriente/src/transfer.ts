export const BANCO_ORIENTE_PARTICIPANT_ID = "bank-oriente";

export type TransferState =
  | "INSTRUCTION_ACCEPTED"
  | "FUNDS_RESERVED"
  | "SETTLED"
  | "CREDIT_PENDING"
  | "COMPLETED";

export interface TransferReadModel {
  transfer_id: string;
  state: TransferState;
  amount_minor: number;
  currency: string;
  source_participant_id: string;
  destination_participant_id: string;
  correlation_id: string;
  causation_id: string | null;
  idempotency_key: string;
  created_at: string;
  settled_at: string | null;
  last_updated_at: string;
  schema_version: string;
  trace_id: string;
  participant_id: string;
}

/** Local fixture shaped like the authorized bank-core read response. */
export const transferFixture: TransferReadModel = {
  transfer_id: "tr_2025_000184",
  state: "COMPLETED",
  amount_minor: 125000,
  currency: "DOP",
  source_participant_id: "bank-costa",
  destination_participant_id: BANCO_ORIENTE_PARTICIPANT_ID,
  correlation_id: "cor_7c1f5e1d8b4a",
  causation_id: "evt_accepted_000184",
  idempotency_key: "idem_tr_2025_000184_9a71",
  created_at: "2025-03-08T14:22:11Z",
  settled_at: "2025-03-08T14:22:16Z",
  last_updated_at: "2025-03-08T14:22:18Z",
  schema_version: "1.0",
  trace_id: "trace_5b8d2f31c9e0",
  participant_id: BANCO_ORIENTE_PARTICIPANT_ID,
};

export const transferStates: readonly TransferState[] = [
  "INSTRUCTION_ACCEPTED",
  "FUNDS_RESERVED",
  "SETTLED",
  "CREDIT_PENDING",
  "COMPLETED",
];

export function isTransferVisibleToParticipant(
  transfer: TransferReadModel,
  participantId: string,
): boolean {
  return (
    transfer.participant_id === participantId &&
    (transfer.source_participant_id === participantId ||
      transfer.destination_participant_id === participantId)
  );
}

export function maskIdempotencyKey(key: string): string {
  if (key.length <= 8) return "••••";
  return `${key.slice(0, 5)}••••${key.slice(-4)}`;
}

export function formatAmount(amountMinor: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amountMinor / 100);
}
