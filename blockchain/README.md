# Blockchain demo commands

Run these commands from project root.

## 1) Start DB + API + Ganache

```bash
docker compose up -d db blockchain api
```

## 2) Install JS deps for Hardhat (one-time)

```bash
cd blockchain
npm install
cd ..
```

## 3) Deploy contract to local chain

```bash
cd blockchain
npx hardhat compile
npx hardhat run scripts/deploy.js --network local
cd ..
```

Copy values from output:

- `BLOCKCHAIN_CONTRACT_ADDRESS=...`
- `BLOCKCHAIN_FROM_ADDRESS=...`

## 4) Enable anchoring in API

Create/update `.env` in project root:

```bash
BLOCKCHAIN_RPC_URL=http://blockchain:8545
BLOCKCHAIN_CONTRACT_ADDRESS=0x...
BLOCKCHAIN_FROM_ADDRESS=0x...
```

Restart API:

```bash
docker compose up -d --build api
```

## 5) Send one measurement

```bash
curl -s -X POST http://localhost:8080/api/measurements \
  -H 'Content-Type: application/json' \
  -d '{"deviceId":"sensor-demo","timestamp":"2026-02-17T12:00:00Z","temperature":18.7,"ph":7.3}'
```

## 6) Verify saved hash and blockchain tx info

```bash
curl -s 'http://localhost:8080/api/measurements?deviceId=sensor-demo&from=2026-02-17T00:00:00Z&to=2026-02-18T00:00:00Z&limit=10'
```

You should see fields:

- `dataHash`
- `anchorTxHash`
- `anchorBlockNumber`

## 7) Optional direct ethers.js call

```bash
cd blockchain
BLOCKCHAIN_RPC_URL=http://127.0.0.1:8545 \
BLOCKCHAIN_CONTRACT_ADDRESS=0x... \
BLOCKCHAIN_PRIVATE_KEY=0x... \
DATA_HASH=0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
node scripts/anchor-demo.js
```
