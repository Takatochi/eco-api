# eco-api demo runbook (for students)

This project demonstrates:

- blockchain data integrity (`dataHash` + on-chain anchor);
- reliability and testing (unit tests + load-like simulator metrics).

## Prerequisites

- Docker + Docker Compose
- Go 1.25+ (for `go test`)
- Node.js 18+ and npm (for local contract deploy)

## Full start (exact command sequence)

1. Start infrastructure and API:

```bash
docker compose up -d db blockchain api
```

2. Install blockchain toolchain (one-time):

```bash
cd blockchain
npm install
```

3. Compile + deploy Solidity contract:

```bash
npx hardhat compile
npx hardhat run scripts/deploy.js --network local
cd ..
```

4. Create `.env` in project root with values from deploy output:

```bash
BLOCKCHAIN_RPC_URL=http://blockchain:8545
BLOCKCHAIN_CONTRACT_ADDRESS=0x...   # from CONTRACT=
BLOCKCHAIN_FROM_ADDRESS=0x...       # from DEPLOYER=
```

5. Restart API with blockchain env:

```bash
docker compose up -d --build api
```

6. Start sensor simulators:

```bash
docker compose up -d --build sensor-1 sensor-2 sensor-3 sensor-4
```

7. Watch logs:

```bash
docker compose logs -f api sensor-1
```

## Quick manual check

Create one measurement:

```bash
curl -s -X POST http://localhost:8080/api/measurements \
  -H 'Content-Type: application/json' \
  -d '{"deviceId":"sensor-demo","timestamp":"2026-02-17T12:00:00Z","temperature":18.7,"ph":7.3}'
```

Read measurements and verify blockchain fields:

```bash
curl -s 'http://localhost:8080/api/measurements?deviceId=sensor-demo&from=2026-02-17T00:00:00Z&to=2026-02-18T00:00:00Z&limit=10'
```

Expected fields in response item:

- `dataHash`
- `anchorTxHash`
- `anchorBlockNumber`

## Reliability/load demonstration

Run one high-load simulator instance:

```bash
docker compose run --rm -e DEVICE_ID=stress-1 -e WORKERS=8 -e INTERVAL_SEC=0.2 sensor-1
```

Watch summary metrics printed by simulator:

- `errorRate`
- `avgLatency`
- `rps`

## Tests

```bash
go test ./...
```

## Stop everything

```bash
docker compose down -v
```

## Shortcuts (Makefile)

```bash
make test
make demo-up
make demo-down
make blockchain-install
make blockchain-deploy
```
