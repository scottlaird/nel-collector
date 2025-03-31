# nel-collector

This is a small agent that listens for HTTP on a port and logs
[NEL](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Network_Error_Logging)
responses to a database.  This can be used to collect client-side HTTP
metrics from browsers, including HTTP errors and timing metrics.

This mostly only works with Chrome-family browsers today; Firefox has
support but it's disabled by default.

This is still a work in progress.

## Installing

You'll need [Go](https://go.dev) 1.21 or higher installed, then just run

```
$ go install github.com/scottlaird/nel-collector@latest
```

This should download and compile the collector code and leave a
`nel-compiler` binary in your Go bin directory, usually `~/go/bin`.

At the moment, it's compiled with Postgresql, MySQL, and Clickhouse
drivers.

### Creating your database schema

See the [schemas/](schemas/) subdirectory.  If you don't see your DB
there, then file an issue and I'll see what I can do to help.

## Running

Flags:

- `-db_table=<tablename>`.  **Required**.  Specify the name of the
  database table that `nel-collector` will write into.  This must
  exist already.
- `-listen=[<host>]:<port>`.  Specify which host and port
  `nel-collector` will use to listen for HTTP traffic.  Defaults to
  `:8080`.
- `-max_message_size=<bytes>`.  Limit the maximum NEL message allowed.
  Defaults to 1 MB.
- `-number_of_proxies=<count>`.  Tells `nel-collector` to extract
  client IPs from the `X-Forwarded-For` header, using the nth header
  from the right.  The default value is 0, which makes `nel-collector`
  ignore the `X-Forwarded-For` header.  Setting it to `1` tells it to
  use the first forwarded IP, `2` uses the second forwarded IP, and so
  forth.
- `-allow_additional_body`.  By default, `nel-collector` only logs
  known fields from the `body` field of the NEL message.  If this flag
  is enabled then unknown fields will be added to the
  `additional_body` column in the database.
- `-read_timeout=<seconds>`, `-write_timeout=<seconds>`.  Set HTTP
  read and write timeouts.  Defaults to 10s each.
- `-tracing`.  Enable OpenTelemetry tracing.

Environment variables:

- `DB_DRIVER=<driver>`.  Sets the database driver to use.  Currently
  valid settings are `clickhouse`, `mysql` and `pgx` (for Postgresql).
- `DSN=<value>`.  Specifies how to connect to your database.
    - For Clickhouse, this should look like
      `clickhouse://<user>:<pass>@<host>:9000/<dbname>"`.  See
      [docs](https://github.com/ClickHouse/clickhouse-go?tab=readme-ov-file#dsn)
    - For Postgres: `user=<user> password=<password> host=<hostname>
      port=<port> database=<database> sslmode=disable`.
    - For MySQL:
      `[tcp:<addr>|unix:<sockpath>]*<dbname>/<user>/<password>` or
      just `<dbname>/<user>/<password>`.

### Logging

`nel-collector` should log errors to STDOUT.

### Tracing

`nel-collector` has partial support for OpenTelemetry tracing, enabled
with `-trace`.  If you don't have an otel collector running on port
4317 locally, then you'll want to set the
`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` environment variable to point to
your collector.


### systemd unit

I should include a systemd unit file here.  TBD.
