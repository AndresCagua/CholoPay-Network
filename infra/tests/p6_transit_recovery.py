"""Docker-only P6 transit recovery probe; explicit validation messages, no product claims."""
import hashlib, hmac, json, os, urllib.error, urllib.request
BASE=os.getenv("TRANSIT_URL","http://transit:8082"); SECRET=os.getenv("TRANSIT_EVENT_SECRET","local-test-only")
STATE_FILE="/tmp/p6-transit-state.json"
def require(condition,message):
    if not condition: raise RuntimeError("P6 transit validation failed: "+message)
def call(method,path,body=None,headers=None):
    data=json.dumps(body).encode() if body is not None else None
    req=urllib.request.Request(BASE+path,data=data,method=method,headers={"Content-Type":"application/json",**(headers or {})})
    try:
        with urllib.request.urlopen(req) as r:return r.status,json.loads(r.read())
    except urllib.error.HTTPError as e:return e.code,json.loads(e.read())
def signed_event(event,event_id):
    raw=json.dumps(event,separators=(",",":")).encode(); sig=hmac.new(SECRET.encode(),raw,hashlib.sha256).hexdigest()
    return call("POST","/v1/internal/events",event,{"X-Transit-Signature":sig})
def first_run():
    status,reg=call("POST","/v1/registrations",{"plate":"P6-SYN-001","vehicle_type":"synthetic-bus"}); require(status==201,"registration creation returned %s: %s"%(status,reg))
    status,fine_result=call("POST","/v1/fines",{"registration_id":reg["id"],"reason":"synthetic P6 validation","amount_minor":2500}); require(status==201,"fine creation returned %s: %s"%(status,fine_result))
    obligations=[reg["obligation_id"],fine_result["obligation"]["id"]]; intents=[]
    for index,oid in enumerate(obligations):
        key="p6-transit-intent-%d"%index; status,intent=call("POST","/v1/payment-intents",{"obligation_id":oid},{"Idempotency-Key":key}); require(status==201,"intent creation returned %s: %s"%(status,intent))
        replay_status,replay=call("POST","/v1/payment-intents",{"obligation_id":oid},{"Idempotency-Key":key}); require(replay_status==200 and replay["id"]==intent["id"],"identical intent replay changed identity")
        conflict_status,conflict=call("POST","/v1/payment-intents",{"obligation_id":oid,"different":True},{"Idempotency-Key":key}); require(conflict_status==409 and conflict.get("error")=="idempotency_key_payload_mismatch","intent fingerprint conflict was not rejected")
        intents.append(intent)
    receipts=[]
    for index,(oid,intent) in enumerate(zip(obligations,intents)):
        event={"event_id":"p6-transit-event-%d"%index,"type":"payment.final.confirmed","payment_intent_id":intent["id"],"obligation_id":oid}
        status,result=signed_event(event,event["event_id"]); require(status==200 and result["status"]=="APPLIED","signed event application returned %s: %s"%(status,result))
        replay_status,replay=signed_event(event,event["event_id"]); require(replay_status==200 and replay==result,"signed event replay changed result")
        receipt_status,receipt=call("GET","/v1/obligations/%s/receipt"%oid); require(receipt_status==200 and receipt.get("synthetic") is True,"synthetic receipt unavailable")
        receipts.append(receipt)
    status,constancia=call("POST","/v1/constancias",{"registration_id":reg["id"]}); require(status==201 and constancia.get("synthetic") is True,"synthetic constancia unavailable")
    state={"registration_id":reg["id"],"obligation_ids":obligations,"intent_ids":[i["id"] for i in intents],"receipt_ids":[r["id"] for r in receipts],"constancia_id":constancia["id"]}
    with open(STATE_FILE,"w",encoding="utf-8") as f:json.dump(state,f)
    print("P6 transit initial state created: "+json.dumps(state,sort_keys=True))
def recovery_run():
    try:
        with open(STATE_FILE,encoding="utf-8") as f:state=json.load(f)
    except OSError as exc: raise RuntimeError("missing first-run state: %s"%exc)
    status,reg=call("GET","/v1/registrations/%s"%state["registration_id"]); require(status==200 and reg["id"]==state["registration_id"],"registration identity did not survive recreation")
    for oid,iid,rid in zip(state["obligation_ids"],state["intent_ids"],state["receipt_ids"]):
        status,obligation=call("GET","/v1/obligations/%s"%oid); require(status==200 and obligation["status"]=="paid","obligation %s did not remain paid"%oid)
        status,intent=call("GET","/v1/payment-intents/%s"%iid); require(status==200 and intent["id"]==iid and intent["status"]=="confirmed","payment intent %s was not recovered"%iid)
        status,receipt=call("GET","/v1/obligations/%s/receipt"%oid); require(status==200 and receipt["id"]==rid,"receipt identity did not survive recreation")
    status,constancia=call("GET","/v1/constancias/%s"%state["constancia_id"]); require(status==200 and constancia["id"]==state["constancia_id"],"constancia identity did not survive recreation")
    print("P6 transit recovery validation: PASS (synthetic IDs, paid statuses, receipts, constancia and replay state survived)")
if os.getenv("P6_TRANSIT_RECOVERY_MODE")=="recovery": recovery_run()
else: first_run()
