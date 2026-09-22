# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Сборка
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /app/a-scam-bot ./cmd/app

# -----------------------------
# Runtime
# -----------------------------

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    tzdata

COPY --from=builder /app/a-scam-bot /app/a-scam-bot

ENV TZ=Europe/Moscow

ENTRYPOINT ["/app/a-scam-bot"]