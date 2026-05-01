FROM golang:1.24-alpine

RUN apk add --no-cache python3

RUN adduser -D -u 1001 runner
USER runner

WORKDIR /workspace
