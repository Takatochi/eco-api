CREATE TABLE IF NOT EXISTS measurements(
    id SERIAL primary key,//
    divice_id text not null,
    ts timestamptz not null,
    temparature double precision
);
create index IF NOT EXISTS
    idx_measurements_device_ts on
        measurements(divice_id, ts desc);
create unique index IF NOT EXISTS
    uq_measurements_device_ts on
        measurements(divice_id, ts);