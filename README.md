# CEX Go (Centralized Exchange Backend)

A backend service for a simplified crypto exchange, built in Go.

It provides:
- user registration and login with JWT
- market listing and market creation
- deposits and balance tracking
- order creation and order book retrieval
- websocket streams for market updates

The project uses PostgreSQL as the source of truth and Redis for fast order book access and real-time pub/sub.

## Table of Contents

- [CEX Go (Centralized Exchange Backend)](#cex-go-centralized-exchange-backend)
  - [Table of Contents](#table-of-contents)
  - [What This Project Is](#what-this-project-is)
  - [Tech Stack](#tech-stack)
  - [System Architecture](#system-architecture)
  - [How the Flow Works](#how-the-flow-works)
    - [1) Startup Flow](#1-startup-flow)
    - [2) Auth + Deposit Flow](#2-auth--deposit-flow)
    - [3) Order Placement Flow (Current)](#3-order-placement-flow-current)
    - [4) Real-Time Market Stream Flow](#4-real-time-market-stream-flow)
  - [API Overview](#api-overview)
    - [Public Endpoints](#public-endpoints)
    - [Auth-Protected Endpoints](#auth-protected-endpoints)
    - [Example Requests](#example-requests)
  - [Quick Start (Docker)](#quick-start-docker)
  - [Run Locally (Without Docker for App)](#run-locally-without-docker-for-app)
  - [Environment Variables](#environment-variables)
  - [WebSocket Usage](#websocket-usage)
  - [Project Structure](#project-structure)
  - [Roadmap Ideas](#roadmap-ideas)

## What This Project Is

This repository is an exchange-style backend where users can:
1. create an account and log in
2. deposit assets (mock deposit)
3. create buy/sell orders
4. read order books per market
5. subscribe to market updates over websockets

It is designed as a practical backend foundation and currently includes important building blocks for matching and real-time updates.

## Tech Stack

- Language: Go 1.25
- HTTP framework: Gin
- Auth: JWT (HMAC)
- Database: PostgreSQL 17
- Cache + Pub/Sub: Redis 7
- Containers: Docker + Docker Compose
- Dev hot reload: Air

Main Go dependencies:
- github.com/gin-gonic/gin
- github.com/golang-jwt/jwt/v5
- github.com/jackc/pgx/v5 (database/sql driver)
- github.com/redis/go-redis/v9
- github.com/joho/godotenv

## System Architecture

```mermaid
flowchart LR
    C[Client App]
    W[WebSocket Client]
    A[Go API - Gin]
    M[Middleware JWT Verify]
    Q[Query Layer]
    P[(PostgreSQL)]
    R[(Redis)]
    S[WS Server Rooms]

    C -->|HTTP REST| A
    W -->|ws /ws/:marketId| S
    A --> M
    M --> Q
    Q --> P
    A --> R
    S -->|Subscribe orderbook/trades channels| R
```

## How the Flow Works

### 1) Startup Flow

```mermaid
sequenceDiagram
    participant App as Go App
    participant PG as PostgreSQL
    participant RD as Redis

    App->>PG: Connect using POSTGRES_URI
    App->>RD: Connect using REDIS_ADDR
    App->>PG: Fetch active open orders
    App->>RD: Load order book snapshot
    App->>App: Register REST + WS routes
    App->>App: Start server (:PORT)
```

### 2) Auth + Deposit Flow

```mermaid
sequenceDiagram
    participant U as User
    participant API as API
    participant JWT as JWT Module
    participant DB as PostgreSQL

    U->>API: POST /register
    API->>DB: Create user
    API-->>U: User ID

    U->>API: POST /login
    API->>DB: Validate email/password
    API->>JWT: Generate token
    API-->>U: JWT token

    U->>API: POST /deposit (Bearer token)
    API->>JWT: Validate token
    API->>DB: Credit balance
    API-->>U: Deposit success
```

### 3) Order Placement Flow (Current)

```mermaid
sequenceDiagram
    participant U as User
    participant API as API
    participant DB as PostgreSQL

    U->>API: POST /orders
    API->>DB: Validate market
    API->>DB: Lock user balance
    API->>DB: Insert order (pending)
    API-->>U: Order ID
```

Note: A matching engine exists in the codebase, but order matching is not yet wired into the create-order request path.

### 4) Real-Time Market Stream Flow

```mermaid
sequenceDiagram
    participant C as Client WS
    participant WS as WS Server
    participant R as Redis

    C->>WS: Connect /ws/:marketId
    WS->>WS: Join/Create room
    WS->>R: Subscribe orderbook:marketId + trades:marketId
    R-->>WS: Pub/Sub messages
    WS-->>C: Broadcast updates

    C->>WS: Disconnect
    WS->>WS: Leave room
    WS->>R: Cancel subscription when room empty
```

## API Overview

Base URL (local): `http://localhost:8080`

### Public Endpoints

- `POST /register`
- `POST /login`
- `GET /markets`
- `GET /markets/:id`
- `POST /markets` (marked admin-only in comments, currently no admin middleware)
- `GET /order_book/:market_id`
- `GET /ws/:marketId` (websocket)

### Auth-Protected Endpoints

- `POST /deposit`
- `GET /balances`

Important: `POST /orders` currently does not enforce auth middleware in routing. It expects `user_id` from context in controller logic.

### Example Requests

Register:

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"123456"}'
```

Login:

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"123456"}'
```

Deposit (replace TOKEN):

```bash
curl -X POST http://localhost:8080/deposit \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"asset":"USDT","amount":1000}'
```

Get balances:

```bash
curl -X GET http://localhost:8080/balances \
  -H "Authorization: Bearer TOKEN"
```

Create order:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "market_id":"BTC-USDT",
    "order_type":"limit",
    "side":"buy",
    "price":100,
    "quantity":0.5
  }'
```

Get order book:

```bash
curl http://localhost:8080/order_book/BTC-USDT
```

## Quick Start (Docker)

Prerequisites:
- Docker
- Docker Compose

Run full stack:

```bash
docker compose up --build
```

Services:
- API: `http://localhost:8080`
- Postgres: `localhost:5433`
- Redis: `localhost:6379`

Stop:

```bash
docker compose down
```

With volumes removed:

```bash
docker compose down -v
```

## Run Locally (Without Docker for App)

1. Start dependencies only (Postgres + Redis)

```bash
docker compose up -d postgres redis
```

2. Create `.env` in project root

```env
POSTGRES_URI=postgres://cex:cex@localhost:5433/cex?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
JWT_SECRET=changeme
PORT=8080
```

3. Run app

```bash
go run ./cmd/main.go
```

Dev hot-reload option:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

## Environment Variables

- `POSTGRES_URI`: PostgreSQL connection string
- `REDIS_ADDR`: Redis host:port
- `REDIS_PASSWORD`: Redis password (if any)
- `JWT_SECRET`: secret used to sign/validate JWT
- `PORT`: API port (default `8080`)

## WebSocket Usage

Connect to:
- `ws://localhost:8080/ws/<marketId>`

Example:
- `ws://localhost:8080/ws/BTC-USDT`

Server behavior:
- one room per market
- Redis subscriptions are active only while at least one client is connected
- broadcasts Redis payloads to all clients in the same room

## Project Structure

```text
cmd/
  main.go                # app entrypoint

db/
  migrations/            # schema migration SQL files
  queries/               # SQL query layer

internal/
  auth/                  # JWT and password helpers
  controllers/           # HTTP handlers
  db/                    # postgres + redis connection and helpers
  matching/              # matching engine logic (not fully wired in request path)
  middleware/            # auth middleware
  models/                # request/domain models
  routes/                # route registration
  ws/                    # websocket room + redis subscription logic
```


## Roadmap Ideas

- add strong password hashing (bcrypt/argon2)
- enforce auth middleware consistently for trading endpoints
- connect matching engine into order placement path
- publish trade events and order updates consistently
- add order cancel endpoint + user order history
- add tests (unit + integration)
- add API documentation (OpenAPI/Swagger)

