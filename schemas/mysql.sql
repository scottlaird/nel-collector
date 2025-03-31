-- DB schema for MySQL (untested)

-- We probably shouldn't be using `text` here.
CREATE TABLE nellog (
       `timestamp` timestamp(6),  -- does MySQL support specifying timezone here?
       `age` bigint,
       `type` text,
       `url` text,
       `hostname` text,
       `client_ip` text,
       `sampling_fraction` real,
       `elapsed_time` real,
       `phase` text,
       `body_type` text,
       `server_ip` text,
       `protocol` text,
       `referrer` text,
       `method` text,
       `request_headers` text,  -- maybe json?
       `response_headers` text, -- maybe json?
       `status_code` int,
       `additional_body` text -- maybe json?
);
