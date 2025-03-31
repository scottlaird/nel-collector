package collector

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HTTP Metrics
var (
	requests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nel_collector_requests",
		Help: "The total number of received HTTP requests",
	})
	readErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nel_collector_read_errors",
		Help: "The number of HTTP requests that failed with read errors",
	})
	truncatedErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nel_collector_truncated_errors",
		Help: "The number of HTTP requests that failed due to truncation for being too large",
	})
	parseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nel_collector_parse_errors",
		Help: "The number of HTTP requests that failed due to JSON parsing errors",
	})
	requestLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "nel_collector_request_latency_seconds",
		Help: "A histogram of request latency",
		// Create buckets from 1ms to 10 seconds, with 10 steps per order of magnitude,
		// or roughly a 25% jump between buckets.
		Buckets: prometheus.ExponentialBucketsRange(0.001, 10.000, 41),
	})
	responseCodes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nel_collector_status_codes",
		Help: "The number of each HTTP status code",
	}, []string{"status_code"})
	requestBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "nel_collector_request_size_bytes",
		Help: "A histogram of request size",
		// Create buckets from 1 byte to 2 MB with 5 steps per order of magnitude,
		// or roughly a 60% jump between buckets.
		Buckets: prometheus.ExponentialBucketsRange(1, 10000000, 7*5+1),
	})
	requestEntries = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "nel_collector_request_size_entries",
		Help: "A histogram of the number of records per request",
		// Create buckets from 1 to 1000 5 steps per order of magnitude,
		// or roughly a 60% jump between buckets.
		Buckets: prometheus.ExponentialBucketsRange(1, 1000, 3*5+1),
	})
)

// NELHandler is a http.Handler that can be used for serving NEL requests.
type NELHandler struct {
	NumberOfProxies     int
	MaxBytes            int64
	AllowAdditionalBody bool
	DB                  DBConfig
}

// MaximumBytes() returns the maximum number of bytes allowed in a
// POST request.  Any requests larger than this will fail and return a
// 413.
func (nh *NELHandler) MaximumBytes() int64 {
	if nh.MaxBytes > 0 {
		return nh.MaxBytes
	} else {
		return 1 << 20 // 1 MB
	}
}

// ServeHTTP handles NEL HTTP requests.
func (nh *NELHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	start := time.Now()
	requests.Inc()

	ctx := req.Context()
	span := trace.SpanFromContext(ctx)
	span.AddEvent("Received request")

	// recordTime updates requestLatency with the time since this request started.
	recordTime := func() {
		elapsed := time.Since(start)
		requestLatency.Observe(elapsed.Seconds())
	}
	// fail handles failures, making sure that the span is
	// updated, an HTTP error is returned, and status code metrics
	// are updated.
	fail := func(status int, err error, msg string) {
		span.RecordError(err)
		span.SetStatus(codes.Error, msg)
		http.Error(resp, msg, status)
		responseCodes.WithLabelValues(fmt.Sprintf("%d", status)).Inc()
		recordTime()
	}

	if req.Method != "POST" {
		fail(405, nil, "POST required")
		return
	}

	cap := nh.MaximumBytes()

	body := bytes.NewBuffer(make([]byte, 0, cap)) // Cap the number of bytes read
	b, err := body.ReadFrom(req.Body)
	if err != nil {
		readErrors.Inc()
		slog.Error("Unable to read from req.Body", "error", err)
		fail(400, err, "Read error")
		return
	}

	requestBytes.Observe(float64(b))

	if b >= cap {
		truncatedErrors.Inc()
		slog.Error("Message truncated", "size", b)
		fail(413, err, "Too big")
		return
	}

	records, err := ParseMessage(body.Bytes())
	if err != nil {
		parseErrors.Inc()
		slog.Error("Unable to parse JSON", "error", err, "json", body.Bytes())
		fail(400, err, "Parse Error")
		return
	}

	var clientIP string
	if nh.NumberOfProxies > 0 {
		ips := req.Header.Get("X-Forwarded-For")
		addresses := strings.Split(ips, ",")
		if ips != "" && len(addresses) >= nh.NumberOfProxies {
			clientIP = strings.TrimSpace(addresses[len(addresses)-nh.NumberOfProxies])
		}
	}
	hostname, _ := os.Hostname()
	outRecords := []NelRecord{}

	for _, record := range records {
		h, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			record.ClientIP = h
		}

		record.ClientIP = clientIP
		record.Hostname = hostname

		// Strip the `AdditionalBody` field unless it's explicitly
		// allowed by flags.
		if !nh.AllowAdditionalBody {
			record.AdditionalBody = nil
		}

		outRecords = append(outRecords, record)
	}

	requestEntries.Observe(float64(len(outRecords)))
	span.AddEvent(fmt.Sprintf("Writing %d records to DB", len(outRecords)))

	err = nh.DB.Write(ctx, outRecords)
	if err != nil {
		slog.Error("Unable to write to DB", "error", err)
		fail(500, err, "DB Error")
		return
	}

	io.WriteString(resp, "OK\n")
	span.SetStatus(codes.Ok, "")

	responseCodes.WithLabelValues("200").Inc()
	recordTime()
}
