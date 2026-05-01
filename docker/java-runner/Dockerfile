FROM eclipse-temurin:21-jdk-alpine

ARG JACKSON_VERSION=2.17.2

RUN apk add --no-cache python3 wget && \
    mkdir -p /opt/jars && \
    cd /opt/jars && \
    wget -q "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-databind/${JACKSON_VERSION}/jackson-databind-${JACKSON_VERSION}.jar" && \
    wget -q "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-core/${JACKSON_VERSION}/jackson-core-${JACKSON_VERSION}.jar" && \
    wget -q "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-annotations/${JACKSON_VERSION}/jackson-annotations-${JACKSON_VERSION}.jar"

RUN adduser -D -u 1001 runner
USER runner

WORKDIR /workspace
