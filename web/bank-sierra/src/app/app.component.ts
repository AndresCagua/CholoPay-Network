import { ChangeDetectionStrategy, Component, OnInit } from "@angular/core";
import { SIERRA_PARTICIPANT_ID, TransferReadModel, TransferState, createTransferApi } from "./transfer-read-model";

const STATE_LABELS: Record<TransferState, string> = {
  INSTRUCTION_ACCEPTED: "Instruction accepted",
  FUNDS_RESERVED: "Funds reserved",
  SETTLED: "SETTLED — network settlement confirmed",
  CREDIT_PENDING: "Credit pending",
  COMPLETED: "COMPLETED — beneficiary credited",
  UNKNOWN: "Unknown state",
};

@Component({
  selector: "sierra-root",
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="shell">
      <header class="hero">
        <div>
          <p class="eyebrow">Participant read-only view</p>
          <h1>Banco Sierra</h1>
          <p class="lede">Transfers visible to <strong>{{ participantId }}</strong></p>
        </div>
        <span class="access-badge">READ_ONLY</span>
      </header>

      <section class="notice" aria-label="Data access boundary">
        <strong>Local contract fixture</strong>
        <span>This panel reads an authorized transfer projection only. It does not access a database or create transfers.</span>
      </section>

      <section aria-labelledby="transfers-heading">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Authorized projection</p>
            <h2 id="transfers-heading">Transfer activity</h2>
          </div>
          <span class="count">{{ visibleTransfers.length }} visible</span>
        </div>

        @if (loading) { <p class="empty" role="status">Loading transfers…</p> }
        @if (error) { <p class="empty" role="alert">{{ error }}</p> }
        @for (transfer of visibleTransfers; track transfer.transfer_id) {
          <article class="transfer-card" [attr.aria-label]="'Transfer ' + transfer.transfer_id">
            <div class="card-topline">
              <div>
                <p class="transfer-id">{{ transfer.transfer_id }}</p>
                <p class="amount">{{ formatAmount(transfer.amount_minor, transfer.currency) }}</p>
              </div>
              <span class="state" [class.state-settled]="transfer.state === 'SETTLED'" [class.state-completed]="transfer.state === 'COMPLETED'">
                {{ stateLabel(transfer.state) }}
              </span>
            </div>

            <dl class="route">
              <div><dt>From</dt><dd>{{ transfer.source_participant_id }}</dd></div>
              <div class="arrow" aria-hidden="true">→</div>
              <div><dt>To</dt><dd>{{ transfer.destination_participant_id }}</dd></div>
            </dl>

            <dl class="metadata">
              <div><dt>Schema version</dt><dd>{{ transfer.schema_version }}</dd></div>
              <div><dt>Participant scope</dt><dd>{{ transfer.participant_id }}</dd></div>
              <div><dt>Correlation ID</dt><dd>{{ transfer.correlation_id }}</dd></div>
              <div><dt>Causation ID</dt><dd>{{ transfer.causation_id ?? "—" }}</dd></div>
              <div><dt>Idempotency key</dt><dd>{{ transfer.idempotency_key }}</dd></div>
              <div><dt>Trace ID</dt><dd>{{ transfer.trace_id }}</dd></div>
              <div><dt>Created (UTC)</dt><dd>{{ transfer.created_at }}</dd></div>
              <div><dt>Settled (UTC)</dt><dd>{{ transfer.settled_at ?? "Not settled" }}</dd></div>
              <div><dt>Last updated (UTC)</dt><dd>{{ transfer.last_updated_at }}</dd></div>
            </dl>
          </article>
        } @empty {
          <p class="empty">No transfers are authorized for this participant.</p>
        }
      </section>
    </main>
  `,
  styles: `
    :host { display: block; min-height: 100vh; color: #17252b; background: #f5f7f6; font-family: Inter, ui-sans-serif, system-ui, sans-serif; }
    .shell { max-width: 980px; margin: 0 auto; padding: 3.5rem 1.5rem 5rem; }
    .hero, .section-heading, .card-topline { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
    .hero { margin-bottom: 2rem; }
    .eyebrow { margin: 0 0 .5rem; color: #53746c; font-size: .75rem; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; }
    h1, h2, p { margin-top: 0; }
    h1 { margin-bottom: .5rem; font-size: clamp(2rem, 5vw, 3.5rem); letter-spacing: -.05em; }
    h2 { margin-bottom: 0; font-size: 1.6rem; letter-spacing: -.03em; }
    .lede { color: #647276; }
    .access-badge, .count, .state { border-radius: 999px; font-size: .72rem; font-weight: 800; letter-spacing: .05em; }
    .access-badge { padding: .55rem .75rem; color: #1d5d4e; background: #d9efe8; }
    .notice { display: flex; flex-wrap: wrap; gap: .5rem 1rem; margin-bottom: 3rem; padding: 1rem 1.25rem; border-left: 4px solid #319878; background: white; box-shadow: 0 8px 24px #1d3e3510; }
    .notice span { color: #647276; }
    .section-heading { align-items: end; margin-bottom: 1rem; }
    .count { padding: .4rem .65rem; color: #53746c; background: #e5eeeb; }
    .transfer-card { margin-bottom: 1.25rem; padding: 1.5rem; border: 1px solid #dbe5e1; border-radius: 1rem; background: #fff; box-shadow: 0 10px 28px #1d3e350c; }
    .transfer-id { margin-bottom: .5rem; color: #647276; font-family: ui-monospace, monospace; font-size: .78rem; }
    .amount { margin-bottom: 0; font-size: 1.7rem; font-weight: 800; letter-spacing: -.04em; }
    .state { max-width: 290px; padding: .65rem .8rem; color: #4d5f64; background: #edf1f0; text-align: right; }
    .state-settled { color: #185d93; background: #e2f0fb; }
    .state-completed { color: #1d6b4f; background: #dcf3e9; }
    .route { display: grid; grid-template-columns: 1fr auto 1fr; gap: 1rem; align-items: end; margin: 1.5rem 0; padding: 1rem 0; border-top: 1px solid #edf1f0; border-bottom: 1px solid #edf1f0; }
    .route dt, .metadata dt { margin-bottom: .3rem; color: #7a888b; font-size: .72rem; font-weight: 700; text-transform: uppercase; }
    .route dd, .metadata dd { margin: 0; overflow-wrap: anywhere; font-family: ui-monospace, monospace; font-size: .82rem; }
    .arrow { color: #319878; font-size: 1.4rem; }
    .metadata { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1.1rem; margin: 0; }
    .empty { padding: 2rem; border: 1px dashed #b8c9c3; border-radius: 1rem; color: #647276; text-align: center; }
    @media (max-width: 680px) { .hero, .card-topline { flex-direction: column; } .state { max-width: none; text-align: left; } .metadata { grid-template-columns: 1fr 1fr; } }
    @media (max-width: 420px) { .shell { padding-inline: 1rem; } .metadata { grid-template-columns: 1fr; } }
  `,
})
export class AppComponent implements OnInit {
  readonly participantId = SIERRA_PARTICIPANT_ID;
  visibleTransfers: TransferReadModel[] = [];
  loading = true;
  error: string | null = null;

  ngOnInit(): void {
    createTransferApi(this.participantId).read().then((transfers) => {
      this.visibleTransfers = transfers;
      this.loading = false;
    }).catch((reason: unknown) => {
      this.error = reason instanceof Error ? reason.message : "Transfer API unavailable";
      this.loading = false;
    });
  }

  stateLabel(state: TransferState): string {
    return STATE_LABELS[state];
  }

  formatAmount(amountMinor: number, currency: string): string {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency,
      minimumFractionDigits: 2,
    }).format(amountMinor / 100);
  }
}
