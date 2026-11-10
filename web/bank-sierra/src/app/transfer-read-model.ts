export const SIERRA_PARTICIPANT_ID = "bank-sierra" as const;

export type TransferState =
  | "INSTRUCTION_ACCEPTED"
  | "FUNDS_RESERVED"
  | "SETTLED"
  | "CREDIT_PENDING"
  | "COMPLETED"
  | "UNKNOWN";

export interface TransferReadModel {
  transfer_id: string;
  state: TransferState;
  amount_minor: number;
  currency: string;
  source_participant_id: string;
  destination_participant_id: string;
  schema_version: string;
  correlation_id: string;
  causation_id: string | null;
  idempotency_key: string;
  created_at: string;
  settled_at: string | null;
  last_updated_at: string;
  trace_id: string;
  participant_id: string;
}

/** Local contract fixture; production data is supplied by the Sierra backend. */
export const transferFixtures: readonly TransferReadModel[] = [
  {
    transfer_id: "trf_sierra_2025_0001",
    state: "SETTLED",
    amount_minor: 125000,
    currency: "MXN",
    source_participant_id: SIERRA_PARTICIPANT_ID,
    destination_participant_id: "bank-costa",
    schema_version: "1.0",
    correlation_id: "cor_sierra_0001",
    causation_id: "evt_instruction_0001",
    idempotency_key: "idem_••••••••89ab",
    created_at: "2025-03-08T14:20:00Z",
    settled_at: "2025-03-08T14:20:04Z",
    last_updated_at: "2025-03-08T14:20:04Z",
    trace_id: "trace_sierra_0001",
    participant_id: SIERRA_PARTICIPANT_ID,
  },
  {
    transfer_id: "trf_sierra_2025_0002",
    state: "COMPLETED",
    amount_minor: 87500,
    currency: "MXN",
    source_participant_id: "bank-costa",
    destination_participant_id: SIERRA_PARTICIPANT_ID,
    schema_version: "1.0",
    correlation_id: "cor_sierra_0002",
    causation_id: "evt_credit_pending_0002",
    idempotency_key: "idem_••••••••34cd",
    created_at: "2025-03-08T15:02:00Z",
    settled_at: "2025-03-08T15:02:03Z",
    last_updated_at: "2025-03-08T15:02:08Z",
    trace_id: "trace_sierra_0002",
    participant_id: SIERRA_PARTICIPANT_ID,
  },
  {
    transfer_id: "trf_costa_2025_0099",
    state: "FUNDS_RESERVED",
    amount_minor: 49900,
    currency: "MXN",
    source_participant_id: "bank-costa",
    destination_participant_id: "bank-oriente",
    schema_version: "1.0",
    correlation_id: "cor_costa_0099",
    causation_id: null,
    idempotency_key: "idem_••••••••7788",
    created_at: "2025-03-08T15:30:00Z",
    settled_at: null,
    last_updated_at: "2025-03-08T15:30:01Z",
    trace_id: "trace_costa_0099",
    participant_id: "bank-costa",
  },
];
