# C4 containers

```mermaid
flowchart LR
  customer["Synthetic customer"] --> bankA["Bank Costa API\nSpring Boot"]
  customer --> bankB["Bank Sierra API\nSpring Boot"]
  customer --> bankC["Bank Oriente API\nSpring Boot"]
  bankA <-->|mTLS / signed messages| network["CholoNet\nGo modular monolith"]
  bankB <-->|mTLS / signed messages| network
  bankC <-->|mTLS / signed messages| network
  bankA --> dbA[(PostgreSQL Costa)]
  bankB --> dbB[(PostgreSQL Sierra)]
  bankC --> dbC[(PostgreSQL Oriente)]
  network --> dbN[(PostgreSQL network + settlement)]
  network --> redis[(Redis cache / ephemeral state)]
  network --> kafka[(Kafka optional events)]
  gateway["Gateway\nNode.js / TypeScript"] --> network
  billers["Synthetic billers"] --> gateway
  operator["Operator projections"] --> network
  kafka --> projections["Audit / notification projections"]
```

Each PostgreSQL database is a distinct service and volume. Redis is never authoritative for balances, ledger entries, final settlement state, or the only copy of a key. Kafka is not a source of truth.
