# Vision and boundaries

CholoPay demonstrates interoperable instant-payment workflows with three independent simulated banks, a central network, a directory of CholoKeys, and a settlement ledger. All money, identities, accounts, obligations, and balances are synthetic.

## Non-goals

No real banking, Pix/SPI/DICT interoperability, regulatory certification, real personal data, card PAN/CVV infrastructure, or paid/closed dependency is in scope.

## Phase 0 boundary

Phase 0 establishes written architecture, contracts, threat model, local FOSS infrastructure, health checks, and reproducible smoke verification. It does not implement customer-facing interfaces or financial behavior.

## Trust boundaries

Banks own customer accounts and bank ledgers. The network owns participant registration, CholoKey directory, settlement accounts, and settlement state. Facturadores own obligations. The gateway never writes a ledger. Operator projections are read-only unless an explicitly audited administrative operation is introduced by ADR.
