-- DB schema for Clickhouse

set enable_json_type=1;
CREATE OR REPLACE TABLE nellog (
       `timestamp` DateTime64(6, 'UTC'),
       `age` UInt64,
       `type` String,
       `url` String,
       `hostname` String,
       `client_ip` String,
       `sampling_fraction` Float64,
       `elapsed_time` Float64,
       `phase` String,
       `body_type` String,
       `server_ip` String,
       `protocol` String,
       `referrer` String,
       `method` String,
       `request_headers` String,
       `response_headers` String,
       `status_code` UInt16,
       `additional_body` String
) ENGINE = MergeTree
ORDER BY tuple(timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAYS DELETE;
