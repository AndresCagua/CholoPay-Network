import { useEffect, useState } from "react";
import { BANCO_ORIENTE_PARTICIPANT_ID, createTransferApi, formatAmount, maskIdempotencyKey, transferStates, type TransferReadModel } from "./transfer";

const stateLabels: Record<TransferReadModel["state"], string> = {
  INSTRUCTION_ACCEPTED: "INSTRUCTION_ACCEPTED",
  FUNDS_RESERVED: "FUNDS_RESERVED",
  SETTLED: "SETTLED",
  CREDIT_PENDING: "CREDIT_PENDING",
  COMPLETED: "COMPLETED",
};

function App() {
  const [transfers, setTransfers] = useState<TransferReadModel[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => { createTransferApi(BANCO_ORIENTE_PARTICIPANT_ID).read().then(setTransfers).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "Transfer API unavailable")); }, []);
  const transfer = transfers?.[0] ?? null;

  return (
    <main>
      <header className="hero">
        <div>
          <p className="eyebrow">CholoPay Network · Banco Oriente</p>
          <h1>Transfer visibility</h1>
          <p className="intro">
            Read-only view for transfers authorized for participant{" "}
            <code>{BANCO_ORIENTE_PARTICIPANT_ID}</code>.
          </p>
        </div>
        <span className="readonly-badge">READ_ONLY</span>
      </header>

      {error ? <section role="alert"><h2>Transfer unavailable</h2><p>{error}</p></section> : transfers === null ? <section role="status"><h2>Loading transfers…</h2></section> : transfer ? (
        <>
          <section aria-labelledby="state-heading" className="state-panel">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Transfer {transfer.transfer_id}</p>
                <h2 id="state-heading">Current state</h2>
              </div>
              <strong className="current-state">{stateLabels[transfer.state]}</strong>
            </div>
            <ol className="state-list" aria-label="Transfer state progression">
              {transferStates.map((state) => (
                <li
                  className={state === transfer.state ? "active" : ""}
                  key={state}
                >
                  <span className="state-dot" aria-hidden="true" />
                  <span>{stateLabels[state]}</span>
                </li>
              ))}
            </ol>
            <p className="state-note">
              <strong>SETTLED</strong> confirms network settlement. It is distinct
              from <strong>COMPLETED</strong>, which confirms the bank credit.
            </p>
          </section>

          <section aria-labelledby="transfer-heading">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Authorized read model</p>
                <h2 id="transfer-heading">Transfer details</h2>
              </div>
              <span className="amount">
                {formatAmount(transfer.amount_minor, transfer.currency)}
              </span>
            </div>
            <dl className="details-grid">
              <div><dt>Source participant</dt><dd>{transfer.source_participant_id}</dd></div>
              <div><dt>Destination participant</dt><dd>{transfer.destination_participant_id}</dd></div>
              <div><dt>Currency</dt><dd>{transfer.currency}</dd></div>
              <div><dt>Created at (UTC)</dt><dd>{transfer.created_at}</dd></div>
              <div><dt>Settled at (UTC)</dt><dd>{transfer.settled_at ?? "Not settled"}</dd></div>
              <div><dt>Last updated (UTC)</dt><dd>{transfer.last_updated_at}</dd></div>
            </dl>
          </section>

          <section aria-labelledby="metadata-heading">
            <p className="eyebrow">Traceability</p>
            <h2 id="metadata-heading">Contract metadata</h2>
            <dl className="metadata-grid">
              <div><dt>schema_version</dt><dd>{transfer.schema_version}</dd></div>
              <div><dt>correlation_id</dt><dd>{transfer.correlation_id}</dd></div>
              <div><dt>causation_id</dt><dd>{transfer.causation_id ?? "—"}</dd></div>
              <div><dt>idempotency_key</dt><dd>{maskIdempotencyKey(transfer.idempotency_key)}</dd></div>
              <div><dt>trace_id</dt><dd>{transfer.trace_id}</dd></div>
              <div><dt>participant_id</dt><dd>{transfer.participant_id}</dd></div>
            </dl>
          </section>

          <p className="fixture-note">Participant-scoped API read only · no database access</p>
        </>
      ) : (
        <section role="status"><h2>Transfer unavailable</h2></section>
      )}
    </main>
  );
}

export default App;
