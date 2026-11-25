"""Minimal test consumer: commit Kafka only after the PostgreSQL HTTP boundary accepts."""
import hashlib, hmac, json, os, time, urllib.error, urllib.request
import psycopg
from kafka import KafkaConsumer

TOPIC = os.getenv("KAFKA_TOPIC", "cholonet.inter-network.v1")
BOOTSTRAP = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "kafka:9092")
NETWORK_URL = os.getenv("NETWORK_URL", "http://network:8080/internal/v1/inter-network/transfers")
SECRET = os.environ["CHOLONET_HMAC_SECRET"]

def deliver(body):
    request_id = body["event_id"]
    timestamp = str(int(time.time()))
    signature = hmac.new(SECRET.encode(), f"{timestamp}.{request_id}.".encode() + body["_raw"], hashlib.sha256).hexdigest()
    request = urllib.request.Request(NETWORK_URL, data=body["_raw"], method="POST", headers={
        "Content-Type": "application/json", "X-CholoPay-Timestamp": timestamp,
        "X-CholoPay-Request-Id": request_id, "X-CholoPay-Signature": signature,
    })
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            return response.status
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")
        raise RuntimeError(f"boundary returned HTTP {error.code}: {detail}") from error

def main():
    consumer = KafkaConsumer(TOPIC, bootstrap_servers=BOOTSTRAP, group_id="cholonet-network-boundary",
                             enable_auto_commit=False, auto_offset_reset="earliest",
                             key_deserializer=lambda value: value.decode())
    record = next(consumer)
    raw = record.value
    event = json.loads(raw)
    if record.key != event["event_id"]:
        raise SystemExit(f"stable Kafka key mismatch: {record.key} != {event['event_id']}")
    event["_raw"] = raw
    first = deliver(event)
    second = deliver(event)  # intentional at-least-once replay; boundary must deduplicate
    if first not in (200, 202) or second != 200:
        raise SystemExit(f"boundary acknowledgement mismatch: first={first} second={second}")
    with psycopg.connect(os.environ["NETWORK_DB_URL"]) as database:
        count = database.execute("SELECT COUNT(*) FROM inter_network_message WHERE message_id = %s", (event["payload"]["transfer_id"],)).fetchone()[0]
        if count != 1:
            raise SystemExit(f"durable message count changed: {count}")
    consumer.commit()
    print(json.dumps({"event_id": event["event_id"], "key": record.key, "first_status": first,
                      "replay_status": second, "topic": record.topic, "partition": record.partition,
                      "offset": record.offset}, sort_keys=True))

if __name__ == "__main__":
    main()
