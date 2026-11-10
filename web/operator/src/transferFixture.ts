export type TransferState =
  | "ACCEPTED"
  | "RESERVED"
  | "SETTLED"
  | "CREDIT_PENDING"
  | "COMPLETED"
  | "UNKNOWN";

export type OperatorRole =
  | "NETWORK_ADMIN"
  | "SETTLEMENT_OPERATOR"
  | "AUDITOR"
  | "READ_ONLY";

export interface TransferReadModel {
  transfer_id: string;
  state: TransferState;
  amount_minor: number;
  currency: string;
  source_participant_id: string;
  destination_participant_id: string;
  correlation_id: string;
  causation_id: string;
  idempotency_key: string;
  created_at: string;
  settled_at: string | null;
  last_updated_at: string;
  schema_version: string;
  trace_id: string;
  participant_id: string;
}

export interface OperatorIdentity {
  subject: string;
  participant_id: string;
  roles: readonly OperatorRole[];
  scopes: readonly string[];
  issuer: string;
  audience: string;
  issued_at: string;
  expires_at: string;
}

export const operatorIdentity: OperatorIdentity = {
  subject: "operator.settlement.demo",
  participant_id: "cholonet",
  roles: ["SETTLEMENT_OPERATOR"],
  scopes: ["transfers:read", "settlement:read"],
  issuer: "cholopay-local-docker",
  audience: "cholonet-api",
  issued_at: "2025-03-01T09:00:00Z",
  expires_at: "2025-03-01T17:00:00Z",
};

export const transferFixture: TransferReadModel = {
  transfer_id: "trf_01JNW8JQ4T8K7M2N3P4Q5R6S7T",
  state: "SETTLED",
  amount_minor: 125000,
  currency: "BOB",
  source_participant_id: "bank-costa",
  destination_participant_id: "bank-sierra",
  correlation_id: "corr_01JNW8JQ4T8K7M2N3P4Q5R6S7U",
  causation_id: "evt_01JNW8JQ4T8K7M2N3P4Q5R6S7V",
  idempotency_key: "idem_••••••••••••7f2a",
  created_at: "2025-03-01T10:14:00Z",
  settled_at: "2025-03-01T10:14:08Z",
  last_updated_at: "2025-03-01T10:14:09Z",
  schema_version: "transfer.read.v1",
  trace_id: "trace_01JNW8JQ4T8K7M2N3P4Q5R6S7W",
  participant_id: "cholonet",
};

export const transferStateLabels: Record<TransferState, string> = {
  ACCEPTED: "Instruction accepted",
  RESERVED: "Funds reserved",
  SETTLED: "SETTLED — network settlement confirmed",
  CREDIT_PENDING: "Credit pending",
  COMPLETED: "COMPLETED — beneficiary credited",
  UNKNOWN: "Unknown state",
};

export const readOnlyRoles: readonly OperatorRole[] = [
  "NETWORK_ADMIN",
  "SETTLEMENT_OPERATOR",
  "AUDITOR",
  "READ_ONLY",
];
