DROP TABLE IF EXISTS measurements;

CREATE TABLE IF NOT EXISTS measurements (
    id BIGSERIAL PRIMARY KEY,
    device_id TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    temperature DOUBLE PRECISION,
    ph DOUBLE PRECISION,
    turbidity DOUBLE PRECISION,
    conductivity DOUBLE PRECISION,
    data_hash TEXT NOT NULL,
    anchor_tx_hash TEXT,
    anchor_block_number BIGINT
);

CREATE INDEX IF NOT EXISTS idx_measurements_device_ts
    ON measurements(device_id, ts DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_measurements_device_ts
    ON measurements(device_id, ts);
