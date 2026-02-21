# eco-api

A REST API for ingesting and querying environmental sensor measurements, with optional blockchain anchoring for data integrity.

Built with **Go**, **PostgreSQL**, **Gin**, and **Hardhat / Solidity**.

---

## About

eco-api collects water quality measurements from IoT sensors (temperature, pH, turbidity, conductivity) and stores them in PostgreSQL. Each measurement gets a deterministic SHA-256 hash. Optionally, that hash is anchored on-chain via a Solidity smart contract — providing tamper-evident proof that the data existed at a specific block.

**Architecture**

```
HTTP (Gin)
    └── Handler
         └── Service  (validation · hashing · anchoring)
              └── Repository  (PostgreSQL via pgx v5)
              └── Blockchain   (Ethereum JSON-RPC)
```

**Key properties**

- Blockchain anchoring is fully optional — the API works without it
- Anchor failures never break measurement ingestion (best-effort, logged)
- Duplicate measurements (same device + timestamp) are rejected with 409
- Structured JSON logging via `log/slog`
- Clean architecture with interface-based dependency injection (fully testable)

**Stack**

| Layer | Technology |
|---|---|
| API | Go 1.25, Gin |
| Database | PostgreSQL 16, pgx v5 |
| Blockchain | Ganache (local), Hardhat, Solidity 0.8.24 |
| Simulator | Python 3.12, multi-threaded |
| Container | Docker, distroless runtime image |

---

## Prerequisites

| Tool | Version |
|---|---|
| Docker + Docker Compose | any recent |
| Go | 1.25+ |
| Node.js + npm | 18+ (blockchain only) |

---

## Running

### Option 1 — Local development (recommended)

Starts only PostgreSQL in Docker, runs the API with `go run` locally.

```bash
./scripts/run.sh
```

The script waits for Postgres to be healthy before starting the API.
It also loads a `.env` file from the project root if one exists.

```
API:     http://localhost:8080
Swagger: http://localhost:8080/swagger/index.html
```

### Option 2 — Full Docker stack

Starts DB, blockchain node, API, and all 4 sensor simulators.

```bash
./scripts/run.sh --docker
# or
make demo-up
```

### Option 3 — With blockchain anchoring

```bash
# 1. Start infrastructure
./scripts/run.sh --docker

# 2. Deploy the smart contract (one-time)
make blockchain-install
make blockchain-deploy
# Output includes CONTRACT= and DEPLOYER= addresses

# 3. Create .env with the deployed addresses
cp .env.example .env
# Fill in BLOCKCHAIN_CONTRACT_ADDRESS and BLOCKCHAIN_FROM_ADDRESS

# 4. Restart API to pick up the new env
docker compose up -d --build api
```

### Stop everything

```bash
./scripts/run.sh --down
# or
make demo-down
```

---

## Testing

### Unit tests

```bash
make test
# or
go test ./...
```

Covers: handler HTTP status codes, service validation, hash calculation, anchor success/failure resilience, config validation, Ethereum address format checks.

### End-to-end API tests (bash script)

Requires the API to be running (`./scripts/run.sh`).

```bash
./scripts/test-api.sh
```

The script tests every endpoint and scenario:

| Section | What is tested |
|---|---|
| Health & Ping | `/ping`, `/health`, `/health/ping` |
| OpenAPI | spec is served and valid |
| POST happy path | all fields, required-only fields |
| POST validation | missing deviceId, missing timestamp, invalid timestamp, ph out of range, temperature out of range, invalid JSON |
| POST duplicate | same device + timestamp → 409 |
| POST error safety | response body does not leak postgres internals |
| GET query | results returned, dataHash present, created record found |
| GET validation | missing params, invalid from/to, `to` before `from`, limit out of range |
| GET empty | unknown device returns `[]` |

Optional: point it at any other environment:

```bash
./scripts/test-api.sh http://staging.example.com
```

### Load test (simulator)

```bash
# High-throughput single device
docker compose run --rm \
  -e DEVICE_ID=stress-1 \
  -e WORKERS=8 \
  -e INTERVAL_SEC=0.2 \
  sensor-1
```

The simulator prints periodic metrics: `errorRate`, `avgLatency`, `rps`.

---

## API reference

| Method | Path | Description |
|---|---|---|
| `GET` | `/ping` | Always 200 |
| `GET` | `/health` | 200 if DB is reachable, 503 otherwise |
| `GET` | `/openapi.json` | OpenAPI 3.0 spec |
| `GET` | `/swagger/*` | Swagger UI |
| `POST` | `/api/measurements` | Ingest a measurement |
| `GET` | `/api/measurements` | Query measurements by device and time range |

Full interactive docs at `http://localhost:8080/swagger/index.html`.

### POST /api/measurements

```json
{
  "deviceId": "sensor-1",
  "timestamp": "2026-02-21T10:00:00Z",
  "temperature": 18.5,
  "ph": 7.2,
  "turbidity": 1.1,
  "conductivity": 450.0
}
```

Only `deviceId` and `timestamp` (RFC3339) are required. Sensor fields are optional.

Validation rules:
- `ph` must be in `[0..14]`
- `temperature` must be in `[-50..80]`

### GET /api/measurements

```
GET /api/measurements?deviceId=sensor-1&from=2026-02-21T00:00:00Z&to=2026-02-21T23:59:59Z&limit=100
```

| Parameter | Required | Default | Notes |
|---|---|---|---|
| `deviceId` | yes | — | |
| `from` | yes | — | RFC3339 |
| `to` | yes | — | RFC3339, must be after `from` |
| `limit` | no | 500 | Max 5000 |

---

## Makefile targets

```bash
make run              # ./scripts/run.sh
make run-docker       # ./scripts/run.sh --docker
make run-down         # ./scripts/run.sh --down
make test             # go test ./...
make fmt              # gofmt -w .
make lint             # go vet ./...
make tidy             # go mod tidy
make demo-up          # docker compose up -d --build
make demo-down        # docker compose down -v
make blockchain-install   # npm install in blockchain/
make blockchain-deploy    # compile + deploy contract
```

---

## Project structure

```
eco-api/
├── main.go                          # Entry point, wiring
├── internal/
│   ├── apperr/                      # Typed error (ValidationError)
│   ├── config/                      # Env loading + validation
│   ├── handler/                     # HTTP handlers (Gin)
│   ├── model/                       # Request / response / DB structs
│   ├── repository/                  # PostgreSQL queries (pgx v5)
│   ├── router/                      # Route registration
│   ├── service/                     # Business logic, hashing, anchoring
│   ├── blockchain/                  # Ethereum JSON-RPC client
│   └── docs/                        # Embedded OpenAPI spec
├── blockchain/
│   ├── contracts/DataAnchor.sol     # Solidity smart contract
│   └── scripts/                     # Deploy + demo scripts
├── db/
│   └── init.sql                     # PostgreSQL schema
├── simulator/
│   └── sim.py                       # Multi-threaded load simulator
├── scripts/
│   ├── run.sh                       # Dev run script
│   └── test-api.sh                  # End-to-end API tests
├── Dockerfile                       # Multi-stage build (distroless)
├── docker-compose.yml               # Full stack
└── .env.example                     # Environment variable template
```
