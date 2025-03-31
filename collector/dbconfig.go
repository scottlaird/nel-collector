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
	Write(context.Context, []NelRecord) error
	Connect(context.Context) error
}

type SqlDriver struct {
	pool   *sql.DB
	driver string
	dsn    string
	table  string
}

// NewSqlDriver creates a new SqlDriver object for writing to a
// specified table.  It takes the bulk of its config from the
// `DB_DRIVER` and `DSN` environment variables.
func NewSqlDriver(table string) *SqlDriver {
	db := &SqlDriver{
		driver: os.Getenv("DB_DRIVER"),
		dsn:    os.Getenv("DSN"),
		table:  table,
	}

	return db
}

// Connect connects to a database and validates that we're able to
// access it.
func (db *SqlDriver) Connect(ctx context.Context) error {
	pool, err := sql.Open(db.driver, db.dsn)
	if err != nil {
		return fmt.Errorf("Unable to connect to db (driver=%q, dsn=%q): %v", db.driver, db.dsn, err)
	}
	db.pool = pool

	return pool.PingContext(ctx)
}

// Write writes a slice of NelRecords into the database.
func (db *SqlDriver) Write(ctx context.Context, records []NelRecord) error {
	//slog.Info("db.Write", "record", n)  // TODO: put behind a flag

	// the table name comes from a command-line flag, so I'm
	// relatively okay doing string manipulation on the query
	// here.
	//
	// Is there a less ugly way to insert into 18 columns at once?
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

	// Start a transaction
	tx, err := db.pool.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("Unable to begin transaction", "err", err)
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		slog.Error("Unable to prepare statement", "error", err)
		return err
	}

	for _, record := range records {
		// Marshal the 3 JSON columns into strings.  For some DBs,
		// it's possible that using a JSON columntype would make this
		// less useful; that's a matter for further research.
		req_headers, err := json.Marshal(record.RequestHeaders)
		if err != nil {
			slog.Error("Unable to marshal RequestHeaders", "error", err)
			return err
		}
		resp_headers, err := json.Marshal(record.ResponseHeaders)
		if err != nil {
			slog.Error("Unable to marshal ResponseHeaders", "error", err)
			return err
		}
		add_body, err := json.Marshal(record.AdditionalBody)
		if err != nil {
			slog.Error("Unable to marshal AdditionalBody", "error", err)
			return err
		}

		// ...and actually run the INSERT command.
		_, err = stmt.ExecContext(ctx,
			record.Timestamp, record.Age, record.Type, record.URL,
			record.Hostname, record.ClientIP, record.SamplingFraction, record.ElapsedTime,
			record.Phase, record.BodyType, record.ServerIP, record.Protocol,
			record.Referrer, record.Method, record.StatusCode, string(req_headers),
			string(resp_headers), string(add_body))
		if err != nil {
			return fmt.Errorf("Unable to insert: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		return err
	}
	stmt.Close()

	return nil
}
