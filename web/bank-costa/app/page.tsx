import {
  getCostaTransfers,
  stateLabels,
  type TransferReadModel,
} from "./transfers";

function formatAmount(transfer: TransferReadModel) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: transfer.currency,
  }).format(transfer.amount_minor / 100);
}

function Metadata({ transfer }: { transfer: TransferReadModel }) {
  const fields = [
    ["Schema version", transfer.schema_version],
    ["Correlation ID", transfer.correlation_id],
    ["Causation ID", transfer.causation_id ?? "—"],
    ["Idempotency key", transfer.idempotency_key],
    ["Trace ID", transfer.trace_id],
    ["Participant scope", transfer.participant_id],
  ];

  return (
    <dl className="metadata">
      {fields.map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function TransferCard({ transfer }: { transfer: TransferReadModel }) {
  return (
    <article className="transfer-card" aria-labelledby="transfer-heading">
      <div className="card-heading">
        <div>
          <p className="eyebrow">Transfer read model</p>
          <h2 id="transfer-heading">{transfer.transfer_id}</h2>
        </div>
        <span className="state-badge">{stateLabels[transfer.state]}</span>
      </div>

      <div className="amount-row">
        <div>
          <p className="label">Amount</p>
          <p className="amount">{formatAmount(transfer)}</p>
        </div>
        <div>
          <p className="label">Current state</p>
          <p className="state-value">{transfer.state}</p>
        </div>
      </div>

      <div className="timeline" aria-label="Transfer state history">
        {(["INSTRUCTION_ACCEPTED", "RESERVED", "SETTLED", "CREDIT_PENDING", "COMPLETED"] as const).map(
          (state) => (
            <div className={state === transfer.state ? "timeline-item current" : "timeline-item"} key={state}>
              <span className="timeline-dot" aria-hidden="true" />
              <span>{stateLabels[state]}</span>
            </div>
          ),
        )}
      </div>

      <div className="participants">
        <div>
          <p className="label">Source participant</p>
          <strong>{transfer.source_participant_id}</strong>
        </div>
        <div>
          <p className="label">Destination participant</p>
          <strong>{transfer.destination_participant_id}</strong>
        </div>
      </div>

      <section aria-labelledby="metadata-heading">
        <h3 id="metadata-heading">Contract metadata</h3>
        <Metadata transfer={transfer} />
      </section>

      <footer className="timestamps">
        <span>Created {transfer.created_at}</span>
        <span>Settled {transfer.settled_at ?? "not settled"}</span>
        <span>Updated {transfer.last_updated_at}</span>
      </footer>
    </article>
  );
}

export default function Home() {
  const transfers = getCostaTransfers();

  return (
    <main className="shell">
      <header className="page-header">
        <div>
          <p className="brand">CholoPay Network / Costa</p>
          <h1>Transfer visibility</h1>
          <p className="intro">
            Read-only view for transfers in the Banco Costa participant scope.
          </p>
        </div>
        <span className="scope-pill">Scope: bank-costa</span>
      </header>

      <section aria-labelledby="transfers-heading">
        <div className="section-heading">
          <h2 id="transfers-heading">Costa transfers</h2>
          <span>{transfers.length} transfer</span>
        </div>
        {transfers.map((transfer) => (
          <TransferCard key={transfer.transfer_id} transfer={transfer} />
        ))}
      </section>

      <p className="notice">
        Mock read model only. This panel does not access a database or create
        transfers; production data will be fetched from the Costa bank-core API.
      </p>
    </main>
  );
}
