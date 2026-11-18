package com.cholopay.bankcore.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.Base64;
import java.util.HexFormat;
import java.util.Map;
import java.util.UUID;

@Service
public class JdbcCholoKeyService implements CholoKeyService {
    private final JdbcTemplate jdbc; private final CholoKeyDirectoryClient directory; private final ObjectMapper mapper;
    public JdbcCholoKeyService(JdbcTemplate jdbc, CholoKeyDirectoryClient directory, ObjectMapper mapper) { this.jdbc=jdbc; this.directory=directory; this.mapper=mapper; }
    private void own(String participant, String token, UUID account) {
        if (token == null || token.isBlank()) throw new SecurityException();
        Integer count=jdbc.queryForObject("SELECT COUNT(*) FROM ledger_account a JOIN public_customer c ON c.id=a.customer_id WHERE a.id=? AND c.participant_id=? AND c.api_token_hash=?",Integer.class,account,participant,JdbcPublicAccountService.hashToken(token));
        if (count == null || count == 0) throw new SecurityException();
    }
    private CholoKeyDirectoryClient.DirectoryEntry owned(String participant, UUID participantUuid, String token, UUID keyId) {
        CholoKeyDirectoryClient.DirectoryEntry key=directory.get(keyId); if (!participantUuid.equals(key.participantId())) throw new SecurityException();
        own(participant,token,UUID.fromString(key.accountRef())); return key;
    }
    @Override @Transactional public Key create(String participant, UUID participantUuid, UUID network, String token, UUID account, String type, String value, String kind, String idempotency) {
        own(participant,token,account); if (!directory.configured() || network==null || participantUuid==null) throw new IllegalStateException("la configuración del directorio no está disponible");
        var key=directory.create(UUID.randomUUID(),network,participantUuid,type,value,account.toString(),kind==null?"PERSONAL":kind,idempotency);
        return key(key);
    }
    @Override public Key get(String participant, UUID participantUuid, String token, UUID keyId) { return key(owned(participant,participantUuid,token,keyId)); }
    @Override public void transition(String participant, UUID participantUuid, String token, UUID keyId, String action, String idempotency) { var key=owned(participant,participantUuid,token,keyId); directory.transition(keyId,participantUuid,key.accountRef(),action,idempotency); }
    private Key key(CholoKeyDirectoryClient.DirectoryEntry k){ return new Key(k.keyId(),k.keyType(),k.keyValue(),UUID.fromString(k.accountRef()),k.accountKind(),k.status(),k.version()); }
    private String encode(Map<String,Object> payload) { try { return "cholokey.qr.v1."+Base64.getUrlEncoder().withoutPadding().encodeToString(mapper.writeValueAsBytes(payload)); } catch(Exception e){throw new IllegalStateException(e);} }
    @Override @Transactional public String staticQr(String p, UUID pu, String t, UUID id) { var k=owned(p,pu,t,id); if (!"ACTIVE".equals(k.status())) throw new SecurityException(); String payload=encode(Map.of("version","1","kind","STATIC","synthetic",true,"key_id",k.keyId(),"key_type",k.keyType())); jdbc.update("INSERT INTO public_cholokey_qr_static(payload_hash,key_id) VALUES (?,?) ON CONFLICT (payload_hash) DO NOTHING",hash(payload),k.keyId()); return payload; }
    @Override @Transactional public String dynamicQr(String p, UUID pu, String t, UUID id, long amount, String ref, Instant exp) { var k=owned(p,pu,t,id); if (!"ACTIVE".equals(k.status())) throw new SecurityException(); if(amount<=0||ref==null||ref.isBlank()||ref.length()>160||exp==null||!exp.isAfter(Instant.now())) throw new IllegalArgumentException("el importe, referencia y expiración del QR dinámico no son válidos"); UUID payload=UUID.randomUUID(); String encoded=encode(Map.of("version","1","kind","DYNAMIC","synthetic",true,"payload_id",payload,"key_id",k.keyId(),"amount_minor",amount,"reference",ref,"expires_at",exp.toString())); jdbc.update("INSERT INTO public_cholokey_qr_dynamic(payload_id,key_id,payload_hash,amount_minor,reference,expires_at) VALUES (?,?,?,?,?,?)",payload,k.keyId(),hash(encoded),amount,ref,java.sql.Timestamp.from(exp)); return encoded; }
    @Override @Transactional public Verification verifyQr(String payload) { try { if(payload==null||!payload.startsWith("cholokey.qr.v1.")) return new Verification(false,"payload_no_reconocido",null,0,null,null); JsonNode n=mapper.readTree(Base64.getUrlDecoder().decode(payload.substring(15))); UUID id=UUID.fromString(n.get("key_id").asText()); var key=directory.get(id); if(!"ACTIVE".equals(key.status())) return new Verification(false,"key_not_active",id,0,null,null); if(!"DYNAMIC".equals(n.get("kind").asText())) { Integer known=jdbc.queryForObject("SELECT COUNT(*) FROM public_cholokey_qr_static WHERE payload_hash=? AND key_id=?",Integer.class,hash(payload),id); return known != null && known == 1 ? new Verification(true,"static_synthetic",id,0,null,null) : new Verification(false,"payload_no_reconocido",id,0,null,null); } UUID pid=UUID.fromString(n.get("payload_id").asText()); Instant exp=Instant.parse(n.get("expires_at").asText()); if(!exp.isAfter(Instant.now())) return new Verification(false,"payload_expirado",id,0,null,exp); int changed=jdbc.update("UPDATE public_cholokey_qr_dynamic SET consumed_at=CURRENT_TIMESTAMP WHERE payload_id=? AND payload_hash=? AND consumed_at IS NULL AND expires_at>?",pid,hash(payload),java.sql.Timestamp.from(Instant.now())); if(changed!=1) return new Verification(false,"payload_repetido",id,0,null,exp); return new Verification(true,"dynamic_synthetic",id,n.get("amount_minor").asLong(),n.get("reference").asText(),exp); } catch(Exception e){ return new Verification(false,"payload_no_reconocido",null,0,null,null); } }
    private String hash(String s){try{return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(s.getBytes(StandardCharsets.UTF_8)));}catch(Exception e){throw new IllegalStateException(e);}}
}
