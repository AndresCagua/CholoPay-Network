import { useEffect, useState } from "react";
import { BANCO_ORIENTE_PARTICIPANT_ID, createTransferApi, formatAmount, maskIdempotencyKey, transferStates, type TransferReadModel } from "./transfer";

const stateLabels: Record<TransferReadModel["state"], string> = {
  INSTRUCTION_ACCEPTED: "Instrucción aceptada",
  FUNDS_RESERVED: "Fondos reservados",
  SETTLED: "Liquidada",
  CREDIT_PENDING: "Crédito pendiente",
  COMPLETED: "Completada",
};

function App() {
  const [transfers, setTransfers] = useState<TransferReadModel[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => { createTransferApi(BANCO_ORIENTE_PARTICIPANT_ID).read().then(setTransfers).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : "La API de transferencias no está disponible")); }, []);
  const transfer = transfers?.[0] ?? null;

  return (
    <main>
      <header className="hero">
        <div>
          <p className="eyebrow">CholoPay Network · Banco Oriente</p>
          <h1>Visibilidad de transferencias</h1>
          <p className="intro">
            Vista de solo lectura de las transferencias autorizadas para el participante{" "}
            <code>{BANCO_ORIENTE_PARTICIPANT_ID}</code>.
          </p>
        </div>
        <span className="readonly-badge">READ_ONLY</span>
      </header>

      {error ? <section role="alert"><h2>Transferencias no disponibles</h2><p>{error}</p></section> : transfers === null ? <section role="status"><h2>Cargando transferencias…</h2></section> : transfer ? (
        <>
          <section aria-labelledby="state-heading" className="state-panel">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Transferencia {transfer.transfer_id}</p>
                <h2 id="state-heading">Estado actual</h2>
              </div>
              <strong className="current-state">{stateLabels[transfer.state]}</strong>
            </div>
            <ol className="state-list" aria-label="Progresión de estados de la transferencia">
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
              <strong>Liquidada</strong> confirma la liquidación de red. Es distinta
              de <strong>Completada</strong>, que confirma el crédito bancario.
            </p>
          </section>

          <section aria-labelledby="transfer-heading">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Modelo de lectura autorizado</p>
                <h2 id="transfer-heading">Detalles de la transferencia</h2>
              </div>
              <span className="amount">
                {formatAmount(transfer.amount_minor, transfer.currency)}
              </span>
            </div>
            <dl className="details-grid">
              <div><dt>Participante de origen</dt><dd>{transfer.source_participant_id}</dd></div>
              <div><dt>Participante de destino</dt><dd>{transfer.destination_participant_id}</dd></div>
              <div><dt>Moneda</dt><dd>{transfer.currency}</dd></div>
              <div><dt>Creada (UTC)</dt><dd>{transfer.created_at}</dd></div>
              <div><dt>Liquidada (UTC)</dt><dd>{transfer.settled_at ?? "No liquidada"}</dd></div>
              <div><dt>Última actualización (UTC)</dt><dd>{transfer.last_updated_at}</dd></div>
            </dl>
          </section>

          <section aria-labelledby="metadata-heading">
            <p className="eyebrow">Trazabilidad</p>
            <h2 id="metadata-heading">Metadatos del contrato</h2>
            <dl className="metadata-grid">
              <div><dt>schema_version</dt><dd>{transfer.schema_version}</dd></div>
              <div><dt>correlation_id</dt><dd>{transfer.correlation_id}</dd></div>
              <div><dt>causation_id</dt><dd>{transfer.causation_id ?? "—"}</dd></div>
              <div><dt>idempotency_key</dt><dd>{maskIdempotencyKey(transfer.idempotency_key)}</dd></div>
              <div><dt>trace_id</dt><dd>{transfer.trace_id}</dd></div>
              <div><dt>participant_id</dt><dd>{transfer.participant_id}</dd></div>
            </dl>
          </section>

          <p className="fixture-note">API por participante y de solo lectura · sin acceso a bases de datos</p>
        </>
      ) : (
        <section role="status"><h2>Transferencias no disponibles</h2></section>
      )}
    </main>
  );
}

export default App;
