# Operator web panel

The panel reads `GET {VITE_TRANSFER_API_BASE_URL}/v1/participants/{participant_id}/transfers` through the participant-scoped client in `src/transferFixture.ts`. The response is filtered by the requested participant as a defense-in-depth adapter contract; production authorization remains server-side. If no base URL is configured, the client uses its documented synthetic read adapter (`synthetic://transfer-read`) so the public HTTP boundary can be exercised without browser database, ledger, or filesystem access.

Docker check: `docker build -t cholopay-operator-web ./web/operator`.
