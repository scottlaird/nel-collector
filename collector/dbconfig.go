package collector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBConfig interface {
	Write(context.Context, NelRecord) error
	Connect(context.Context) error
}

type SqlDriver struct {
	pool   *sql.DB
	driver string
	dsn    string
	table  string
}

func NewSqlDriver(table string) *SqlDriver {
	db := &SqlDriver{
		driver: os.Getenv("DB_DRIVER"),
		dsn:    os.Getenv("DSN"),
		table:  table,
	}

	return db
}

func (db *SqlDriver) Connect(ctx context.Context) error {
	pool, err := sql.Open(db.driver, db.dsn)
	if err != nil {
		return fmt.Errorf("Unable to connect to db (driver=%q, dsn=%q): %v", db.driver, db.dsn, err)
	}
	db.pool = pool

	return pool.PingContext(ctx)
}

func (db *SqlDriver) Write(ctx context.Context, n NelRecord) error {
	slog.Info("db.Write", "record", n)

	// the table name comes from a command-line flag, so I'm
	// relatively okay doing string manipulation on the query
	// here.
	query := "INSERT INTO " + db.table +
		"(timestamp, age, type, url, " +
		"hostname, client_ip, sampling_fraction, elapsed_time, " +
		"phase, body_type, server_ip, protocol, " +
		"referrer, method, status_code, request_headers, " +
		"response_headers, additional_body) values " +
		"(?, ?, ?, ?, " +
		"?, ?, ?, ?, " +
		"?, ?, ?, ?, " +
		"?, ?, ?, ?, " +
		"?, ?)"
	req_headers, err := json.Marshal(n.RequestHeaders)
	if err != nil {
		slog.Error("Unable to marshal RequestHeaders", "error", err)
		return err
	}
	resp_headers, err := json.Marshal(n.ResponseHeaders)
	if err != nil {
		slog.Error("Unable to marshal ResponseHeaders", "error", err)
		return err
	}
	add_body, err := json.Marshal(n.AdditionalBody)
	if err != nil {
		slog.Error("Unable to marshal AdditionalBody", "error", err)
		return err
	}
	_, err = db.pool.ExecContext(ctx, query,
		n.Timestamp, n.Age, n.Type, n.URL,
		n.Hostname, n.ClientIP, n.SamplingFraction, n.ElapsedTime,
		n.Phase, n.BodyType, n.ServerIP, n.Protocol,
		n.Referrer, n.Method, n.StatusCode, req_headers,
		resp_headers, add_body)
	if err != nil {
		return fmt.Errorf("Unable to insert: %v", err)
	}

	return nil
}
