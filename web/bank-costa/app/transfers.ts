export const COSTA_PARTICIPANT_ID = "bank-costa" as const;

export const TRANSFER_STATES = [
  "INSTRUCTION_ACCEPTED",
  "RESERVED",
  "SETTLED",
  "CREDIT_PENDING",
  "COMPLETED",
  "UNKNOWN",
] as const;

export type TransferState = (typeof TRANSFER_STATES)[number];

export type TransferReadModel = {
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
};

/** Local contract fixture; a real panel would receive this from Costa bank-core. */
export const costaTransferFixture: TransferReadModel = {
  transfer_id: "trf_costa_00042",
  state: "COMPLETED",
  amount_minor: 125000,
  currency: "CRC",
  source_participant_id: COSTA_PARTICIPANT_ID,
  destination_participant_id: "bank-sierra",
  correlation_id: "corr_01JQ7COSTA42",
  causation_id: "evt_01JQ7RESERVED42",
  idempotency_key: "idem_01JQ7••••••••42",
  created_at: "2025-03-18T14:22:11Z",
  settled_at: "2025-03-18T14:22:14Z",
  last_updated_at: "2025-03-18T14:22:15Z",
  schema_version: "1.0",
  trace_id: "trace_01JQ7COSTA42",
  participant_id: COSTA_PARTICIPANT_ID,
};

export const stateLabels: Record<TransferState, string> = {
  INSTRUCTION_ACCEPTED: "Instruction accepted",
  RESERVED: "Funds reserved",
  SETTLED: "Settled",
  CREDIT_PENDING: "Credit pending",
  COMPLETED: "Completed",
  UNKNOWN: "Unknown state",
};

export function getCostaTransfers(): TransferReadModel[] {
  return [costaTransferFixture].filter(
    (transfer) => transfer.participant_id === COSTA_PARTICIPANT_ID,
  );
}
