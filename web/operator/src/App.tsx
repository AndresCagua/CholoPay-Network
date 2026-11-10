import {
  operatorIdentity,
  readOnlyRoles,
  transferFixture,
  transferStateLabels,
  type TransferReadModel,
} from "./transferFixture";

const stateSequence: TransferReadModel["state"][] = [
  "ACCEPTED",
  "RESERVED",
  "SETTLED",
  "CREDIT_PENDING",
  "COMPLETED",
];

function formatAmount(transfer: TransferReadModel) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: transfer.currency,
    minimumFractionDigits: 2,
  }).format(transfer.amount_minor / 100);
}

function App() {
  const canReadTransfer = operatorIdentity.roles.some((role) =>
    readOnlyRoles.includes(role),
  );

  if (!canReadTransfer) {
    return (
      <main>
        <header>
          <p className="eyebrow">CholoPay Network</p>
          <h1>Operator panel</h1>
        </header>
        <section className="notice" role="alert">
          <h2>Transfer unavailable</h2>
          <p>This identity is not authorized to read transfer resources.</p>
        </section>
      </main>
    );
  }

  const transfer = transferFixture;

  return (
    <main>
      <header className="hero">
        <div>
          <p className="eyebrow">CholoPay Network · CholoNet</p>
          <h1>Transfer read view</h1>
          <p className="status">Backend contract fixture · read-only</p>
        </div>
        <div className="identity" aria-label="Current operator identity">
          <span className="role-badge">{operatorIdentity.roles.join(" · ")}</span>
          <span>{operatorIdentity.subject}</span>
          <span>Scope: {operatorIdentity.participant_id}</span>
        </div>
      </header>

      <section className="summary" aria-labelledby="transfer-heading">
        <div>
          <p className="eyebrow">Transfer</p>
          <h2 id="transfer-heading">{transfer.transfer_id}</h2>
          <p className="route">
            {transfer.source_participant_id} <span aria-hidden="true">→</span>{" "}
            {transfer.destination_participant_id}
          </p>
        </div>
        <div className="amount-block">
          <strong>{formatAmount(transfer)}</strong>
          <span className="state-pill">{transfer.state}</span>
        </div>
      </section>

      <section aria-labelledby="lifecycle-heading">
        <div className="section-heading">
          <h2 id="lifecycle-heading">Transfer lifecycle</h2>
          <span className="read-only">Read-only view</span>
        </div>
        <ol className="timeline">
          {stateSequence.map((state) => (
            <li className={state === transfer.state ? "current" : ""} key={state}>
              <span className="timeline-dot" aria-hidden="true" />
              <span>
                <strong>{state}</strong>
                <small>{transferStateLabels[state]}</small>
              </span>
              {state === transfer.state && <em>current</em>}
            </li>
          ))}
        </ol>
        <p className="hint">
          SETTLED confirms network settlement. COMPLETED is a separate state and
          means the beneficiary credit has completed.
        </p>
      </section>

      <section aria-labelledby="metadata-heading">
        <h2 id="metadata-heading">Contract metadata</h2>
        <dl className="metadata">
          <div><dt>Schema version</dt><dd>{transfer.schema_version}</dd></div>
          <div><dt>Correlation ID</dt><dd>{transfer.correlation_id}</dd></div>
          <div><dt>Causation ID</dt><dd>{transfer.causation_id}</dd></div>
          <div><dt>Idempotency key</dt><dd>{transfer.idempotency_key}</dd></div>
          <div><dt>Trace ID</dt><dd>{transfer.trace_id}</dd></div>
          <div><dt>Participant scope</dt><dd>{transfer.participant_id}</dd></div>
          <div><dt>Created at (UTC)</dt><dd>{transfer.created_at}</dd></div>
          <div><dt>Settled at (UTC)</dt><dd>{transfer.settled_at ?? "Not settled"}</dd></div>
          <div><dt>Last updated (UTC)</dt><dd>{transfer.last_updated_at}</dd></div>
        </dl>
      </section>
    </main>
  );
}

export default App;
