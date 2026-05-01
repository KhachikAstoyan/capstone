# Stage 1: compile the in-VM fc-agent for linux/amd64
FROM --platform=linux/amd64 golang:1.24-alpine AS agent-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/fc-agent/ ./cmd/fc-agent/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /fc-agent ./cmd/fc-agent/

# Stage 2: Python 3 runtime rootfs
FROM --platform=linux/amd64 alpine:3.21
RUN apk add --no-cache python3 && \
    ln -sf python3 /usr/bin/python

# fc-agent + init
COPY --from=agent-builder /fc-agent /usr/local/bin/fc-agent
COPY docker/firecracker/init /sbin/init
RUN chmod +x /sbin/init /usr/local/bin/fc-agent

RUN mkdir -p /workspace /tmp
