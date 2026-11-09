# Phase 2 backlog

| ID | Ticket | Depends on | Acceptance |
|---|---|---|---|
| P2-01 | Adopt reservation/settlement/reconciliation protocol | Phase 1 | ADR 0006 and state contract reviewed |
| P2-02 | Scaffold Go network service and Docker build | P2-01 | health endpoint and container test pass |
| P2-03 | Persist transfer instructions and transitions | P2-02 | invalid transitions and duplicate IDs rejected |
| P2-04 | Add participant and settlement-account schema | P2-03 | no bank-table access; isolated migration |
| P2-05 | Implement central balanced settlement journal | P2-04 | one journal per transfer and conserved positions |
| P2-06 | Add bank adapter contracts and outbox/inbox | P2-05 | at-least-once replay with one financial effect |
| P2-07 | Fault/reconciliation scenarios | P2-06 | issuer/receiver crash states recover without duplicates |
| P2-08 | Integrate CholoKey directory | P2-03 | minimized resolution and uniqueness constraints |
