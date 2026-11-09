# syntax=docker/dockerfile:1
FROM maven:3.9-eclipse-temurin-21 AS test
WORKDIR /workspace
COPY pom.xml .
RUN mvn -q -DskipTests dependency:go-offline
COPY src ./src
RUN mvn -q test

FROM test AS package
RUN mvn -q -DskipTests package

FROM eclipse-temurin:21-jre AS runtime
WORKDIR /app
COPY --from=package /workspace/target/bank-core-0.1.0-SNAPSHOT.jar /app/bank-core.jar
USER 10001
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/bank-core.jar"]
