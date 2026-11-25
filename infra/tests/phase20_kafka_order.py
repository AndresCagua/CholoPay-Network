"""Create one Costa outbox event while the Kafka broker is intentionally stopped."""
import json, os, urllib.request, uuid

body = {
    "sourceAccountId": "00000000-0000-4000-8000-000000000001",
    "destinationAccountId": "00000000-0000-4000-8000-000000000002",
    "sourceParticipantId": "00000000-0000-4000-8000-000000000101",
    "destinationParticipantId": "00000000-0000-4000-8000-000000000102",
    "sourceNetworkId": "00000000-0000-4000-8000-000000000201",
    "destinationNetworkId": "00000000-0000-4000-8000-000000000202",
    "amountMinor": 125,
    "currency": "USD",
    "correlationId": str(uuid.uuid4()),
    "causationId": str(uuid.uuid4()),
    "traceId": "phase20-kafka",
}
request = urllib.request.Request(
    "http://bank-core:8080/internal/v1/inter-network/transfers",
    data=json.dumps(body).encode(), method="POST",
    headers={"Content-Type": "application/json", "Idempotency-Key": "phase20-kafka-" + str(uuid.uuid4())},
)
with urllib.request.urlopen(request, timeout=10) as response:
    result = json.loads(response.read())
print(json.dumps({"event_id": result["eventId"], "transfer_id": result["transferId"]}, sort_keys=True))
