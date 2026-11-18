package com.cholopay.bankcore.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.annotation.JsonProperty;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Map;
import java.util.UUID;

@Component
public class CholoKeyDirectoryClient {
    private final HttpClient http = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3)).build();
    private final ObjectMapper mapper; private final String baseUrl; private final String token; private final String participantToken;
    public CholoKeyDirectoryClient(ObjectMapper mapper,@Value("${CHOLONET_DIRECTORY_URL:}") String baseUrl,@Value("${DIRECTORY_PORT_TOKEN:}") String token,@Value("${DIRECTORY_PARTICIPANT_TOKEN:}") String participantToken) { this.mapper=mapper; this.baseUrl=baseUrl==null?"":baseUrl.replaceAll("/$",""); this.token=token; this.participantToken=participantToken; }
    public boolean configured(){return !baseUrl.isBlank()&&token!=null&&!token.isBlank()&&participantToken!=null&&!participantToken.isBlank();}
    public DirectoryEntry create(UUID keyId,UUID networkId,UUID participantId,String keyType,String keyValue,String accountRef,String accountKind,String idempotency){return request("POST","/keys",Map.of("key_id",keyId,"network_id",networkId,"participant_id",participantId,"key_type",keyType,"key_value",keyValue,"account_ref",accountRef,"account_kind",accountKind),idempotency,DirectoryEntry.class,participantToken);}
    public DirectoryEntry get(UUID keyId){return request("GET","/keys/"+keyId,null,null,DirectoryEntry.class);}
    public void transition(UUID keyId,UUID participantId,String accountRef,String action,String idempotency){request("POST","/keys/"+keyId+"/"+action,Map.of("participant_id",participantId,"account_ref",accountRef,"reason","gestión pública sintética","idempotency_key",idempotency),null,Void.class);}
    private <T> T request(String method,String path,Object body,String idempotency,Class<T> type){ return request(method,path,body,idempotency,type,token); }
    private <T> T request(String method,String path,Object body,String idempotency,Class<T> type,String authToken){
        if(!configured()) throw new IllegalStateException("el directorio CholoKey no está configurado");
        try { var builder=HttpRequest.newBuilder(URI.create(baseUrl+path)).timeout(Duration.ofSeconds(5)).header("Authorization","Bearer "+authToken); if(idempotency!=null) builder.header("Idempotency-Key",idempotency); var publisher=body==null?HttpRequest.BodyPublishers.noBody():HttpRequest.BodyPublishers.ofString(mapper.writeValueAsString(body)); var response=http.send(builder.method(method,publisher).header("Content-Type","application/json").build(),HttpResponse.BodyHandlers.ofString()); if(response.statusCode()>=400) throw new DirectoryException(response.statusCode(),response.body()); return type==Void.class?null:mapper.readValue(response.body(),type); } catch(DirectoryException e){throw e;} catch(Exception e){throw new DirectoryException(503,"el directorio CholoKey no está disponible");}
    }
    public record DirectoryEntry(@JsonProperty("key_id") UUID keyId,@JsonProperty("network_id") UUID networkId,@JsonProperty("participant_id") UUID participantId,@JsonProperty("key_type") String keyType,@JsonProperty("key_value") String keyValue,@JsonProperty("account_ref") String accountRef,@JsonProperty("account_kind") String accountKind,String status,long version) {}
    public static class DirectoryException extends RuntimeException { public final int status; public DirectoryException(int status,String message){super(message);this.status=status;} }
}
