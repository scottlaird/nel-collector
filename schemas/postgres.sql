-- DB schema for Postgres (untested)

CREATE TABLE nellog (
       `timestamp` timestamp (6) with time zone,
       `age` bigint,
       `type` text,
       `url` text,
       `hostname` text,
       `client_ip` text,
       `sampling_fraction` numeric,
       `elapsed_time` numeric,
       `phase` text,
       `body_type` text,
       `server_ip` text,
       `protocol` text,
       `referrer` text,
       `method` text,
       `request_headers` text,
       `response_headers` text,
       `status_code` int,
       `additional_body` text
);
