-- DB schema for Clickhouse

-- This uses `lowCardinality()` on fields where we can expect a
-- relatively low number of distinct values; this changes Clickhouse's
-- encoding (see
-- https://clickhouse.com/docs/sql-reference/data-types/lowcardinality)
-- to gain efficiency when we expect fewer than 10k different values
-- in the field.
set enable_json_type=1;
CREATE OR REPLACE TABLE nellog (
       `timestamp` DateTime64(6, 'UTC') CODEC(Delta, ZSTD),  -- Store the time as compressed deltas.
       `age` UInt64,
       `type` LowCardinality(String),
       `url` String,
       `hostname` LowCardinality(String),  -- the server that runs nel-collector
       `client_ip` String,
       `sampling_fraction` Float32, -- CH doens't like LowCardinality(Float32), unfortunately 
       `elapsed_time` UInt32,  -- Number of milliseconds from the start of the fetch until completion/error
       `phase` LowCardinality(String),
       `body_type` LowCardinality(String),
       `server_ip` LowCardinality(String),
       `protocol` LowCardinality(String),
       `referrer` String,
       `method` LowCardinality(String),
       `request_headers` String,
       `response_headers` String,
       `status_code` UInt16,
       `additional_body` String
) ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY tuple(hostname, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAYS DELETE
SETTINGS async_insert=1;
