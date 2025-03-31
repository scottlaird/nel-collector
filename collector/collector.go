package collector

import (
        "bytes"
        "encoding/json"
        "io"
        "log/slog"
        "net"
        "net/http"
        "strings"
        "time"

        "go.opentelemetry.io/otel/codes"
        "go.opentelemetry.io/otel/trace"

)

// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Network_Error_Logging

type NelRecord struct {
        Timestamp time.Time `json:"timestamp"`
        Age       int64     `json:"age"`
        Type      string    `json:"type"`
        URL       string    `json:"url"`
        Hostname  string    `json:"hostname"`
        ClientIP  string    `json:"client_ip"` // populated from X-Forwarded-For and/or the directly connected IP

        // These are all fields in `body` in the spec; I'm hoisting them into the main struct.
        SamplingFraction float64        `json:"sampling_fraction,omit_empty"`
        ElapsedTime      float64        `json:"elapsed_time,omit_empty"`
        Phase            string         `json:"phase,omit_empty"`
        BodyType         string         `json:"body_type,omit_empty"` // The top-level message and the body both have a `type` field, and they're semantically different and both usually provided.
        ServerIP         string         `json:"server_ip,omit_empty"`
        Protocol         string         `json:"protocol,omit_empty"`
        Referrer         string         `json:"referrer,omit_empty"` // Note the correct spelling in NEL, unlike HTTP.
        Method           string         `json:"method,omit_empty"`
        RequestHeaders   map[string]any `json:"request_headers,omitempty"`
        ResponseHeaders  map[string]any `json:"response_headers,omitempty"`
        statusCodeFloat  float64
        StatusCode       int `json:"status_code,omitzero"`

        AdditionalBody map[string]any `json:"body"` // This is really a JSON blob without an required structure.
}

type NelPostFormat struct {
        Age  int64          `json:"age"`
        Type string         `json:"type"`
        URL  string         `json:"url"`
        Body map[string]any `json:"body"`
}

// getAndClear looks inside of np.Body (a map[string]any) to see if
// the specified key exists.  If so, it attempts to cooerce it into
// the correct type for 'val' using a type assertion and copies it
// into val.  If this succeeds, then the key is removed from Body.
func getAndClear[T any](np NelPostFormat, name string, val *T) {
        if v, ok := np.Body[name]; ok {
                if fv, ok := v.(T); ok {
                        *val = fv
                        delete(np.Body, name)
                }
        }
}

// ParseMessage takes a string from a HTTP POST and turns it into a
// NelRecord.  It copies known values from the `Body` map in the
// message into named fields in the NelRecord, leaving unknown fields
// in `AdditionalBody`
func ParseMessage(msg []byte) (NelRecord, error) {
        np := NelPostFormat{}
        err := json.Unmarshal(msg, &np)

        n := NelRecord{
                Timestamp: time.Now(),
                Age:       np.Age,
                Type:      np.Type,
                URL:       np.URL,
        }

        getAndClear(np, "sampling_fraction", &n.SamplingFraction)
        getAndClear(np, "elapsed_time", &n.ElapsedTime)
        getAndClear(np, "phase", &n.Phase)
        getAndClear(np, "type", &n.BodyType)
        getAndClear(np, "server_ip", &n.ServerIP)
        getAndClear(np, "protocol", &n.Protocol)
        getAndClear(np, "referrer", &n.Referrer)
        getAndClear(np, "method", &n.Method)
        getAndClear(np, "request_headers", &n.RequestHeaders)
        getAndClear(np, "response_headers", &n.ResponseHeaders)

        // Status code is an int, but map[string]any from JSON will
        // always see it as a float.
        getAndClear(np, "status_code", &n.statusCodeFloat)
        n.StatusCode = int(n.statusCodeFloat)

        n.AdditionalBody = np.Body

        return n, err
}

// NELHandler is a http.Handler that can be used for serving NEL requests.
type NELHandler struct {
        NumberOfProxies int
        MaxBytes        int64
        DB              DBConfig
}

func (nh *NELHandler) MaximumBytes() int64 {
        if nh.MaxBytes > 0 {
                return nh.MaxBytes
        } else {
                return 1 << 20 // 1 MB
        }
}

// Handle NEL requests.
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

// TODO(): add database config
func NewNELHandler(db DBConfig) *NELHandler {
        return &NELHandler{DB: db}
}
