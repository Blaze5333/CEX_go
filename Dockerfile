# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o cex ./cmd/main.go

# ── Run stage ─────────────────────────────────────────────────────────────────
FROM alpine:3.21 AS runtime

WORKDIR /app

COPY --from=builder /app/cex .
COPY --from=builder /app/db/migrations ./db/migrations
COPY --from=builder /app/web ./web

EXPOSE 8080

CMD ["./cex"]

# ── Dev stage (hot-reload with air) ───────────────────────────────────────────
FROM golang:1.25-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
