package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/scottlaird/nel-collector/collector"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	oltpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

var (
	allowAdditionalBody = flag.Bool("allow_additional_body", false, "Retain unknown `body` fields from clients in the `additional_body` database column?")
	dbTable             = flag.String("db_table", "", "Name of the database table to write to.")
	listenAddr          = flag.String("listen", ":8080", "Port (and optionally host) to listen for HTTP requests on.")
	maxMsgSize          = flag.Int("max_message_size", 1<<20, "Maximum number of bytes allowed in a NEL POST request.")
	metricsListenAddr   = flag.String("metrics_listen", ":18080", "Port (and optionally host) to serve Prometheus metrics")
	numberOfProxies     = flag.Int("number_of_proxies", 0, "Number of HTTP proxies to expect; this controls how client IPs are extracted from X-Forwarded-For headers.")
	readTimeout         = flag.Int("read_timeout", 10, "Seconds to wait for HTTP reads to finish,")
	trace               = flag.Bool("trace", false, "Enable otel tracing.")
	writeTimeout        = flag.Int("write_timeout", 10, "Seconds to wait for HTTP writes to finish.")
)

// initialize the otel trace collecting infrastructure
func initTracer() (*sdktrace.TracerProvider, error) {
	// Create stdout exporter to be able to retrieve
	// the collected spans.
	exporter, err := oltpgrpc.New(context.Background())
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func main() {
	flag.Parse()

	// I don't want to set a default for this in code, so let's
	// fail fast if the DB table name isn't specified.
	if *dbTable == "" {
		fmt.Fprintf(os.Stderr, "Must supply --db_table=<tablename> at a minimum\n")
		os.Exit(1)
	}

	// Set up otel tracing if --trace is on.
	if *trace {
		tp, err := initTracer()
		if err != nil {
			slog.Error("Unable to initialize otel tracer", "error", err)
			os.Exit(1)
		}
		defer func() {
			tp.Shutdown(context.Background())
		}()
	}

	// Start metrics listener iff --metrics_listen is not empty
	if *metricsListenAddr != "" {
		go func() {
			err := collector.RunMetricsServer(*metricsListenAddr)
			if err != nil {
				slog.Error("Unable to start /metrics server on %s: %v", *metricsListenAddr, err)
				os.Exit(1)
			}
		}()
	}

	// Connect to database.  db.Connect should verify the
	// connection, so this should give us an error quickly if
	// something is wrong.
	db := collector.NewSqlDriver(*dbTable)
	err := db.Connect(context.Background())
	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}

	// Set up the NEL handler from our library.
	nelHandler := collector.NewNELHandler(db)
	nelHandler.NumberOfProxies = *numberOfProxies
	nelHandler.MaxBytes = int64(*maxMsgSize)
	nelHandler.AllowAdditionalBody = *allowAdditionalBody

	// If --trace, then wrap the NEL handler in an otel tracing wrapper.
	var handler http.Handler
	handler = nelHandler
	if *trace {
		handler = otelhttp.NewHandler(nelHandler, "nel")
	}

	// Set up HTTP server
	s := &http.Server{
		Addr:           *listenAddr,
		Handler:        handler,
		ReadTimeout:    time.Duration(*readTimeout) * time.Second,
		WriteTimeout:   time.Duration(*writeTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// ...and run.
	slog.Info("Listening", "addr", s.Addr)
	err = s.ListenAndServe()
	if err != nil {
		slog.Error("HTTP server failed to start", "error", err)
		os.Exit(1)
	}
}
