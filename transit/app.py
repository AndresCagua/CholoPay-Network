"""API sintética de tránsito con PostgreSQL aislado opcional para recuperación Docker."""
from __future__ import annotations
import hashlib, hmac, json, os, threading, time, uuid
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import parse_qs, urlparse

EVENT_SECRET = os.environ.get("TRANSIT_EVENT_SECRET", "synthetic-transit-event-secret")

class TransitStore:
    def __init__(self) -> None:
        self._lock = threading.RLock()
        self.url = os.environ.get("DB_URL", "")
        self.connection = None
        self.registrations: dict[str, dict[str, Any]] = {}
        self.fines: dict[str, dict[str, Any]] = {}
        self.obligations: dict[str, dict[str, Any]] = {}
        self.payment_intents: dict[str, dict[str, Any]] = {}
        self.payment_keys: dict[str, tuple[str, str]] = {}
        self.constancias: dict[str, dict[str, Any]] = {}
        self.events: dict[str, dict[str, Any]] = {}
        if self.url:
            import psycopg
            self.connection = psycopg.connect(self.url, autocommit=True)
            with self.connection.cursor() as cur:
                cur.execute("""
                CREATE TABLE IF NOT EXISTS transit_registration (id TEXT PRIMARY KEY, plate TEXT UNIQUE NOT NULL, record JSONB NOT NULL);
                CREATE TABLE IF NOT EXISTS transit_fine (id TEXT PRIMARY KEY, registration_id TEXT NOT NULL, record JSONB NOT NULL);
                CREATE TABLE IF NOT EXISTS transit_obligation (id TEXT PRIMARY KEY, registration_id TEXT NOT NULL, record JSONB NOT NULL);
                CREATE TABLE IF NOT EXISTS transit_payment_intent (id TEXT PRIMARY KEY, obligation_id TEXT NOT NULL, record JSONB NOT NULL);
                CREATE TABLE IF NOT EXISTS transit_payment_key (idempotency_key TEXT PRIMARY KEY, intent_id TEXT NOT NULL, fingerprint TEXT NOT NULL);
                CREATE TABLE IF NOT EXISTS transit_event (event_id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL, result JSONB NOT NULL);
                CREATE TABLE IF NOT EXISTS transit_constancia (id TEXT PRIMARY KEY, registration_id TEXT NOT NULL, obligation_id TEXT, record JSONB NOT NULL);
                CREATE INDEX IF NOT EXISTS transit_obligation_registration_idx ON transit_obligation(registration_id);
                """)

    @property
    def durable(self) -> bool: return self.connection is not None
    def _memory_name(self, table: str) -> str:
        return {"registration":"registrations","fine":"fines","obligation":"obligations","payment_intent":"payment_intents","constancia":"constancias"}.get(table, table)
    def _all(self, table: str) -> dict[str, dict[str, Any]]:
        if not self.durable: return getattr(self, self._memory_name(table))
        with self.connection.cursor() as cur:
            cur.execute(f"SELECT id, record FROM transit_{table}")
            return {row[0]: row[1] for row in cur.fetchall()}
    def _one(self, table: str, item_id: str) -> dict[str, Any] | None:
        if not self.durable: return getattr(self, self._memory_name(table)).get(item_id)
        with self.connection.cursor() as cur:
            cur.execute(f"SELECT record FROM transit_{table} WHERE id=%s", (item_id,)); row = cur.fetchone()
        return row[0] if row else None
    def _put(self, table: str, record: dict[str, Any], extra: tuple[Any, ...] = ()) -> None:
        if not self.durable: getattr(self, self._memory_name(table))[record["id"]] = record; return
        with self.connection.cursor() as cur:
            registration_id = record.get("registration_id")
            if table == "constancia" and not registration_id:
                obligation = self._one("obligation", record["obligation_id"])
                registration_id = obligation["registration_id"] if obligation else "unknown"
            if table == "registration": names, values = "plate", (record["plate"],)
            elif table == "fine": names, values = "registration_id", (record["registration_id"],)
            elif table == "obligation": names, values = "registration_id", (record["registration_id"],)
            elif table == "payment_intent": names, values = "obligation_id", (record["obligation_id"],)
            else: names, values = "registration_id, obligation_id", (registration_id, record.get("obligation_id"))
            placeholders = ",".join(["%s"] * (2 + len(values)))
            cur.execute(f"INSERT INTO transit_{table} (id,{names},record) VALUES ({placeholders})", (record["id"], *values, json.dumps(record)))
    def _update(self, table: str, record: dict[str, Any]) -> None:
        if not self.durable: getattr(self, self._memory_name(table))[record["id"]] = record; return
        with self.connection.cursor() as cur: cur.execute(f"UPDATE transit_{table} SET record=%s WHERE id=%s", (json.dumps(record), record["id"]))
    def _obligations(self) -> dict[str, dict[str, Any]]: return self._all("obligation")
    def _registrations(self) -> dict[str, dict[str, Any]]: return self._all("registration")

    def register(self, body):
        plate = body.get("plate")
        if not isinstance(plate, str) or not plate.strip(): return 422, {"error":"plate_required"}
        plate = plate.strip().upper()
        with self._lock:
            if any(r["plate"] == plate for r in self._registrations().values()): return 409, {"error":"registration_already_exists", "registration": next(r for r in self._registrations().values() if r["plate"] == plate)}
            record = {"id":str(uuid.uuid4()),"plate":plate,"vehicle_type":body.get("vehicle_type","synthetic-car"),"owner_ref":body.get("owner_ref","synthetic-owner"),"status":"active"}
            obligation = {"id":str(uuid.uuid5(uuid.NAMESPACE_URL, f"cholopay:registration:{record['id']}")),"type":"registration","source_id":record["id"],"registration_id":record["id"],"amount_minor":3200,"currency":"SYN","status":"unpaid"}
            record["obligation_id"] = obligation["id"]
            if self.durable:
                with self.connection.transaction(): self._put("registration", record); self._put("obligation", obligation)
            else: self._put("registration", record); self._put("obligation", obligation)
            return 201, record

    def create_fine(self, body):
        registration_id, reason, amount = body.get("registration_id"), body.get("reason"), body.get("amount_minor")
        with self._lock:
            if registration_id not in self._registrations(): return 404, {"error":"registration_not_found"}
            if not isinstance(reason,str) or not reason.strip() or not isinstance(amount,int) or isinstance(amount,bool) or amount <= 0: return 422, {"error":"reason_and_positive_amount_required"}
            fine={"id":str(uuid.uuid4()),"registration_id":registration_id,"reason":reason.strip(),"amount_minor":amount,"currency":"SYN","status":"unpaid"}
            obligation={"id":str(uuid.uuid4()),"type":"fine","source_id":fine["id"],"registration_id":registration_id,"amount_minor":amount,"currency":"SYN","status":"unpaid"}
            if self.durable:
                with self.connection.transaction(): self._put("fine",fine); self._put("obligation",obligation)
            else: self._put("fine",fine); self._put("obligation",obligation)
            return 201,{"fine":fine,"obligation":obligation}

    def payment_intent(self, body, key):
        obligation_id=body.get("obligation_id")
        if not key: return 400,{"error":"idempotency_key_required"}
        fingerprint=hashlib.sha256(json.dumps(body,sort_keys=True).encode()).hexdigest()
        with self._lock:
            obligation=self._one("obligation", obligation_id) if obligation_id else None
            if not obligation: return 404,{"error":"obligation_not_found"}
            if self.durable:
                with self.connection.transaction():
                    with self.connection.cursor() as cur:
                        cur.execute("SELECT intent_id,fingerprint FROM transit_payment_key WHERE idempotency_key=%s FOR UPDATE",(key,)); previous=cur.fetchone()
                    if previous:
                        if previous[1]!=fingerprint:return 409,{"error":"idempotency_key_payload_mismatch"}
                        return 200,self._one("payment_intent",previous[0])
                    if obligation["status"]!="unpaid":return 409,{"error":"obligation_not_payable","status":obligation["status"]}
                    intent={"id":str(uuid.uuid4()),"obligation_id":obligation_id,"amount_minor":obligation["amount_minor"],"currency":obligation["currency"],"status":"pending","debit_attempted":False,"ledger_written":False}
                    self._put("payment_intent",intent)
                    with self.connection.cursor() as cur: cur.execute("INSERT INTO transit_payment_key VALUES (%s,%s,%s)",(key,intent["id"],fingerprint))
                    return 201,intent
            previous=self.payment_keys.get(key)
            if previous:
                if previous[1]!=fingerprint:return 409,{"error":"idempotency_key_payload_mismatch"}
                return 200,self.payment_intents[previous[0]]
            if obligation["status"]!="unpaid":return 409,{"error":"obligation_not_payable","status":obligation["status"]}
            intent={"id":str(uuid.uuid4()),"obligation_id":obligation_id,"amount_minor":obligation["amount_minor"],"currency":obligation["currency"],"status":"pending","debit_attempted":False,"ledger_written":False}
            self.payment_intents[intent["id"]]=intent; self.payment_keys[key]=(intent["id"],fingerprint); return 201,intent

    def obligations_for_plate(self, plate):
        with self._lock:
            registration=next((r for r in self._registrations().values() if r["plate"]==plate.strip().upper()),None)
            if not registration:return 404,{"error":"vehicle_not_found"}
            return 200,{"vehicle":registration,"items":[o for o in self._obligations().values() if o["registration_id"]==registration["id"]]}

    def apply_gateway_event(self, body,event_id,timestamp,signature):
        if not event_id or not timestamp or not signature:return 401,{"error":"invalid_event_signature"}
        canonical=json.dumps(body,sort_keys=True,separators=(",",":")); expected=hmac.new(EVENT_SECRET.encode(),f"{timestamp}.{canonical}".encode(),hashlib.sha256).hexdigest()
        if not hmac.compare_digest(signature,expected):return 401,{"error":"invalid_event_signature"}
        event={"event_id":event_id,"type":body.get("event_type"),"payment_intent_id":body.get("payment_intent_id"),"obligation_id":body.get("obligation_id")}; raw=json.dumps(event,separators=(",",":")).encode(); sig=hmac.new(EVENT_SECRET.encode(),raw,hashlib.sha256).hexdigest(); return self.apply_event(event,sig)

    def apply_event(self, body, signature):
        raw=json.dumps(body,separators=(",",":")).encode(); expected=hmac.new(EVENT_SECRET.encode(),raw,hashlib.sha256).hexdigest()
        if not signature or not hmac.compare_digest(signature.removeprefix("sha256="),expected):return 401,{"error":"invalid_event_signature"}
        event_id=body.get("event_id")
        if not isinstance(event_id,str) or not event_id:return 422,{"error":"event_id_required"}
        fingerprint=hashlib.sha256(raw).hexdigest()
        with self._lock:
            if self.durable:
                with self.connection.transaction():
                    with self.connection.cursor() as cur: cur.execute("SELECT fingerprint,result FROM transit_event WHERE event_id=%s FOR UPDATE",(event_id,)); old=cur.fetchone()
                    if old:
                        if old[0]!=fingerprint:return 409,{"error":"event_id_reused_with_different_payload"}
                        return 200,old[1]
                    intent=self._one("payment_intent",body.get("payment_intent_id")); obligation=self._one("obligation",body.get("obligation_id"))
                    if not intent or not obligation or intent["obligation_id"]!=obligation["id"]:return 404,{"error":"payment_intent_or_obligation_not_found"}
                    if body.get("type") not in ("payment.final.confirmed","payment.confirmed","payment.settled"):return 422,{"error":"unsupported_event_type"}
                    if body.get("application_available",True) is False: intent["status"]="settled_awaiting_application"; result={"event_id":event_id,"status":"SETTLED_AWAITING_APPLICATION","payment_intent_id":intent["id"],"obligation_id":obligation["id"]}
                    else: intent["status"]="confirmed"; intent["debit_attempted"]=True; obligation["status"]="paid"; result={"event_id":event_id,"status":"APPLIED","payment_intent_id":intent["id"],"obligation_id":obligation["id"]}
                    self._update("payment_intent",intent); self._update("obligation",obligation)
                    with self.connection.cursor() as cur: cur.execute("INSERT INTO transit_event VALUES (%s,%s,%s)",(event_id,fingerprint,json.dumps(result)))
                    return 200,result
            if event_id in self.events:return 200,self.events[event_id]
            intent=self.payment_intents.get(body.get("payment_intent_id")); obligation=self.obligations.get(body.get("obligation_id"))
            if not intent or not obligation or intent["obligation_id"]!=obligation["id"]:return 404,{"error":"payment_intent_or_obligation_not_found"}
            if body.get("type") not in ("payment.final.confirmed","payment.confirmed","payment.settled"):return 422,{"error":"unsupported_event_type"}
            if body.get("application_available",True) is False:intent["status"]="settled_awaiting_application"; result={"event_id":event_id,"status":"SETTLED_AWAITING_APPLICATION","payment_intent_id":intent["id"],"obligation_id":obligation["id"]}
            else:intent["status"]="confirmed";intent["debit_attempted"]=True;obligation["status"]="paid";result={"event_id":event_id,"status":"APPLIED","payment_intent_id":intent["id"],"obligation_id":obligation["id"]}
            self.events[event_id]=result;return 200,result

    def constancia(self, body):
        registration_id=body.get("registration_id")
        with self._lock:
            if registration_id not in self._registrations():return 404,{"error":"registration_not_found"}
            obligations=[o for o in self._obligations().values() if o["registration_id"]==registration_id]
            if not obligations or any(o["status"]!="paid" for o in obligations):return 409,{"error":"constancia_not_available_until_final_confirmation"}
            record={"id":str(uuid.uuid4()),"registration_id":registration_id,"obligation_status":"clear","obligation_ids":[o["id"] for o in obligations],"synthetic":True}
            if self.durable:
                with self.connection.transaction():self._put("constancia",record)
            else:self.constancias[record["id"]]=record
            return 201,record
    def receipt(self, obligation_id):
        with self._lock:
            obligation=self._one("obligation",obligation_id)
            if not obligation:return 404,{"error":"obligation_not_found"}
            if obligation["status"]!="paid":return 409,{"error":"receipt_not_available_until_final_confirmation"}
            existing=next((r for r in self._all("constancia").values() if r.get("obligation_id")==obligation_id),None)
            if existing:return 200,existing
            record={"id":str(uuid.uuid4()),"obligation_id":obligation_id,"amount_minor":obligation["amount_minor"],"currency":obligation["currency"],"synthetic":True}
            if self.durable:
                with self.connection.transaction():self._put("constancia",record)
            else:self.constancias[record["id"]]=record
            return 200,record

STORE=TransitStore()
class TransitHandler(BaseHTTPRequestHandler):
    server_version="CholoPay-Transit-Synthetic/0.2"
    def _json(self,status,body):
        encoded=json.dumps(body,separators=(",",":")).encode();self.send_response(status);self.send_header("Content-Type","application/json");self.send_header("Content-Length",str(len(encoded)));self.end_headers();self.wfile.write(encoded)
    def _body(self):
        value=json.loads(self.rfile.read(int(self.headers.get("Content-Length","0"))) or b"{}");return value if isinstance(value,dict) else {}
    def do_GET(self):
        parsed=urlparse(self.path);path=parsed.path.rstrip("/") or "/"
        if path=="/health/live":return self._json(200,{"status":"ok","service":"transit-synthetic","real_authority":False})
        if path=="/health/ready":return self._json(200,{"status":"ok","service":"transit-synthetic","real_authority":False})
        parts=path.split("/")
        if len(parts)==5 and parts[1:3]==["v1","vehicles"] and parts[4]=="obligations":s,r=STORE.obligations_for_plate(parts[3]);return self._json(s,r)
        if len(parts)==5 and parts[1:3]==["v1","obligations"] and parts[4]=="receipt":s,r=STORE.receipt(parts[3]);return self._json(s,r)
        if len(parts)==4 and parts[1:3]==["v1","registrations"]:
            item=STORE._one("registration",parts[3]) or next((r for r in STORE._registrations().values() if r["plate"]==parts[3].upper()),None);return self._json(200 if item else 404,item or {"error":"registration_not_found"})
        collections={"fines":"fine","obligations":"obligation","payment-intents":"payment_intent","constancias":"constancia"}
        if len(parts)==4 and parts[1:3]==["v1",parts[2]] and parts[2] in collections:
            item=STORE._one(collections[parts[2]],parts[3]);return self._json(200 if item else 404,item or {"error":parts[2].replace("-","_")+"_not_found"})
        if path=="/v1/obligations":
            q=parse_qs(parsed.query);rid=q.get("registration_id",[None])[0];status=q.get("status",[None])[0];items=[o for o in STORE._obligations().values() if (not rid or o["registration_id"]==rid) and (not status or o["status"]==status)];return self._json(200,{"items":items})
        if path=="/v1/status":
            rid=parse_qs(parsed.query).get("registration_id",[None])[0];reg=STORE._one("registration",rid) if rid else None
            if not reg:return self._json(404,{"error":"registration_not_found"})
            return self._json(200,{"registration":reg,"obligations":[o for o in STORE._obligations().values() if o["registration_id"]==rid]})
        return self._json(404,{"error":"not_found"})
    def do_POST(self):
        path=urlparse(self.path).path.rstrip("/")
        try:body=self._body()
        except (ValueError,json.JSONDecodeError):return self._json(400,{"error":"invalid_json"})
        if path=="/v1/registrations":s,r=STORE.register(body)
        elif len(path.split("/"))==5 and path.split("/")[1:3]==["v1","obligations"] and path.split("/")[4]=="payment-intents":body["obligation_id"]=path.split("/")[3];s,r=STORE.payment_intent(body,self.headers.get("Idempotency-Key"))
        elif path=="/v1/internal/events":s,r=STORE.apply_event(body,self.headers.get("X-Transit-Signature") or self.headers.get("X-Event-Signature"))
        elif path=="/v1/internal/payment-events":s,r=STORE.apply_gateway_event(body,self.headers.get("X-CholoPay-Event-Id"),self.headers.get("X-CholoPay-Timestamp"),self.headers.get("X-CholoPay-Signature"))
        elif path=="/v1/fines":s,r=STORE.create_fine(body)
        elif path=="/v1/payment-intents":s,r=STORE.payment_intent(body,self.headers.get("Idempotency-Key"))
        elif path=="/v1/constancias":s,r=STORE.constancia(body)
        else:return self._json(404,{"error":"not_found"})
        return self._json(s,r)
    def log_message(self,format,*args):
        if os.environ.get("ACCESS_LOG","0")=="1":super().log_message(format,*args)
def create_server():return ThreadingHTTPServer((os.environ.get("HOST","0.0.0.0"),int(os.environ.get("PORT","8082"))),TransitHandler)
if __name__=="__main__":create_server().serve_forever()
