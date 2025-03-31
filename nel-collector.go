package main

import (
        "context"
        "flag"
        "log/slog"
        "net/http"
        "os"
        "time"

        "github.com/scottlaird/nel-collector/collector"

        "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
        "go.opentelemetry.io/otel"
        stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
        "go.opentelemetry.io/otel/propagation"
        "go.opentelemetry.io/otel/sdk/resource"
        sdktrace "go.opentelemetry.io/otel/sdk/trace"
        semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

var (
        listenAddr          = flag.String("listen", ":8080", "Port (and optionally host) to listen for HTTP requests on.")
        readTimeout         = flag.Int("read_timeout", 10, "Seconds to wait for HTTP reads to finish,")
        writeTimeout        = flag.Int("write_timeout", 10, "Seconds to wait for HTTP writes to finish.")
        maxMsgSize          = flag.Int("max_message_size", 1<<20, "Maximum number of bytes allowed in a NEL POST request.")
        numberOfProxies     = flag.Int("number_of_proxies", 0, "Number of HTTP proxies to expect; this controls how client IPs are extracted from X-Forwarded-For headers.")
        allowAdditionalBody = flag.Bool("allow_additional_body", false, "Retain unknown `body` fields from clients in the `additional_body` database column?")
        dbTable             = flag.String("db_table", "nellog", "Name of the database table to write to.")
)

func initTracer() (*sdktrace.TracerProvider, error) {
        // Create stdout exporter to be able to retrieve
        // the collected spans.
        exporter, err := stdout.New(stdout.WithPrettyPrint())
        if err != nil {
                return nil, err
        }

        tp := sdktrace.NewTracerProvider(
                sdktrace.WithSampler(sdktrace.AlwaysSample()),
                sdktrace.WithBatcher(exporter),
                sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName("ExampleService"))),
        )
        otel.SetTracerProvider(tp)
        otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
        return tp, nil
}

func main() {
        flag.Parse()

        // Set up otel tracing
        tp, err := initTracer()
        if err != nil {
                slog.Error("Unable to initialize otel tracer", "error", err)
                os.Exit(1)
        }
        defer func() {
                tp.Shutdown(context.Background())
        }()

        db := collector.NewSqlDriver(*dbTable)
        err = db.Connect(context.Background())
        if err != nil {
                slog.Error("Unable to connect to database", "error", err)
                os.Exit(1)
        }

        nelHandler := collector.NewNELHandler(db)
        nelHandler.NumberOfProxies = *numberOfProxies
        nelHandler.MaxBytes = int64(*maxMsgSize)

        s := &http.Server{
                Addr:           *listenAddr,
                Handler:        otelhttp.NewHandler(nelHandler, "foo"),
                ReadTimeout:    time.Duration(*readTimeout) * time.Second,
                WriteTimeout:   time.Duration(*writeTimeout) * time.Second,
                MaxHeaderBytes: 1 << 20,
        }

        slog.Info("Listening", "addr", s.Addr)
        err = s.ListenAndServe()
        if err != nil {
                panic(err)
        }
}
