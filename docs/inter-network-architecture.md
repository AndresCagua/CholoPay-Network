# CholoPay Inter-Network architecture

```mermaid
flowchart TB
  subgraph BANKING[CholoPay Banking Network]
    Costa[Banco Costa]
    Sierra[Banco Sierra]
    Oriente[Banco Oriente]
    BankDirectory[Distributed banking directory]
  end
  subgraph FINTECH[CholoPay Fintech Network]
    WalletA[Wallet A]
    WalletB[Wallet B]
    FintechDirectory[Distributed fintech directory]
  end
  Costa --> BankDirectory
  Sierra --> BankDirectory
  Oriente --> BankDirectory
  WalletA --> FintechDirectory
  WalletB --> FintechDirectory
  BankDirectory --> Router[CholoNet Inter-Network Router]
  FintechDirectory --> Router
  Router --> Settlement[Central simulated settlement ledger]
  Router --> Reconciliation[Cross-network reconciliation]
```

## Route contract

`INTRA_NETWORK` stays within the participant domain. `INTER_NETWORK` crosses domains and requires CholoNet coordination. A missing, suspended, or ambiguous participant is rejected before funds are reserved. A route decision never means `SETTLED`; only the central ledger commit can establish that state.

Every route message includes `schema_version`, `message_type`, `source_network_id`, `destination_network_id`, `source_participant_id`, `destination_participant_id`, `correlation_id`, `causation_id`, `idempotency_key`, and UTC timestamps.

## Simulated CholoKey directory

Each network owns its directory namespace. A lookup is scoped by `network_id`, `key_type`, and `key_value`; the same key value may exist in different network directories without crossing ownership boundaries. The internal resolver is:

```text
POST /internal/v1/directory/resolve
```

A successful response emits a versioned `DIRECTORY_KEY_RESOLVED` evidence envelope. Missing or suspended keys are rejected, and the database uniqueness constraint prevents two active interpretations of the same key inside one network. This is a deterministic simulation, not a connection to a real national directory.
