package collector

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NELHandler is a http.Handler that can be used for serving NEL requests.
type NELHandler struct {
	NumberOfProxies int
	MaxBytes        int64
	DB              DBConfig
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
	ctx := req.Context()
	span := trace.SpanFromContext(ctx)
	span.AddEvent("Received request")

	slog.Info("span.IsRecording", "value", span.IsRecording())

	if req.Method != "POST" {
		http.Error(resp, "POST required", 405)
	}

	cap := nh.MaximumBytes()

	body := bytes.NewBuffer(make([]byte, 0, cap)) // Cap the number of bytes read
	b, err := body.ReadFrom(req.Body)
	if err != nil {
		slog.Error("Unable to read from req.Body", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Unable to read from req.Body")
		http.Error(resp, "Read error", 400)
		return
	}
	if b >= cap {
		slog.Error("Message truncated")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Message truncated")
		http.Error(resp, "Too big", 413)
		return
	}

	record, err := ParseMessage(body.Bytes())
	if err != nil {
		slog.Error("Unable to parse JSON", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "JSON parse failed")
		http.Error(resp, "Parse error", 400)
		return
	}

	h, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		record.ClientIP = h
	}

	if nh.NumberOfProxies > 0 {
		ips := req.Header.Get("X-Forwarded-For")
		addresses := strings.Split(ips, ",")
		if ips != "" && len(addresses) >= nh.NumberOfProxies {
			record.ClientIP = strings.TrimSpace(addresses[len(addresses)-nh.NumberOfProxies])
		}
	}

	span.AddEvent("Writing to DB")

	err = nh.DB.Write(ctx, record)
	if err != nil {
		slog.Error("Unable to write to DB", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB error")
		http.Error(resp, "DB error", 500)
		return
	}

	io.WriteString(resp, "OK\n")
	span.SetStatus(codes.Ok, "")
}
