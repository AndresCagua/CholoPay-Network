package com.cholopay.bankcore.outbox;

import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.common.serialization.StringSerializer;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.Properties;
import java.util.UUID;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/** Minimal Kafka adapter: it publishes the already persisted envelope and never creates ledger state. */
@Component
@ConditionalOnProperty(name = "cholopay.outbox.transport", havingValue = "kafka")
public class KafkaOutboxClient implements OutboxClient, AutoCloseable {
    private static final Pattern EVENT_ID = Pattern.compile("\\\"event_id\\\"\\s*:\\s*\\\"([^\\\"]+)\\\"");
    private volatile KafkaProducer<String, String> producer;
    private final String bootstrapServers;
    private final String clientId;
    private final String topic;

    public KafkaOutboxClient(@Value("${cholopay.outbox.kafka.bootstrap-servers:}") String bootstrapServers,
                             @Value("${cholopay.outbox.kafka.topic:cholonet.inter-network.v1}") String topic,
                             @Value("${cholopay.outbox.kafka.client-id:bank-core-outbox}") String clientId) {
        if (bootstrapServers.isBlank()) throw new IllegalArgumentException("KAFKA_BOOTSTRAP_SERVERS es obligatorio con OUTBOX_TRANSPORT=kafka");
        this.bootstrapServers = bootstrapServers;
        this.clientId = clientId;
        this.topic = topic;
    }

    @Override
    public DeliveryResult send(String serializedEnvelope) {
        Matcher matcher = EVENT_ID.matcher(serializedEnvelope);
        if (!matcher.find()) return DeliveryResult.CONTRACT_FAILURE;
        try {
            producer().send(new ProducerRecord<>(topic, matcher.group(1), serializedEnvelope)).get(Duration.ofSeconds(5).toMillis(), java.util.concurrent.TimeUnit.MILLISECONDS);
            return DeliveryResult.SUCCESS;
        } catch (Exception e) {
            return DeliveryResult.AMBIGUOUS_FAILURE;
        }
    }

    private KafkaProducer<String, String> producer() {
        KafkaProducer<String, String> current = producer;
        if (current != null) return current;
        synchronized (this) {
            if (producer == null) {
                Properties properties = new Properties();
                properties.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
                properties.put(ProducerConfig.CLIENT_ID_CONFIG, clientId);
                properties.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
                properties.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
                properties.put(ProducerConfig.ACKS_CONFIG, "all");
                producer = new KafkaProducer<>(properties);
            }
            return producer;
        }
    }

    @Override public void close() {
        KafkaProducer<String, String> current = producer;
        if (current != null) current.close(Duration.ofSeconds(5));
    }
}
