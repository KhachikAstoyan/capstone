ARG GO_VERSION=1.24.12

FROM golang:${GO_VERSION}-bookworm AS build

ARG SERVICE

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN test -n "${SERVICE}" && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/app /app/app
COPY internal/api/migrations /app/internal/api/migrations
COPY internal/controlplane/migrations /app/internal/controlplane/migrations

USER nonroot:nonroot

ENTRYPOINT ["/app/app"]
