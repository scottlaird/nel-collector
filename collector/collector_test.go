package collector

import (
        "testing"

        "github.com/google/go-cmp/cmp"
        "github.com/google/go-cmp/cmp/cmpopts"
)

func compareNelRecord(t *testing.T, got, want NelRecord) {
        if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(NelRecord{}, "Timestamp"), cmpopts.IgnoreUnexported(NelRecord{})); diff != "" {
                t.Errorf("NelRecord mismatch (-want +got):\n%s", diff)
        }
}

func TestParseString_Simple(t *testing.T) {
        msg := []byte(`{
                         "age": 0,
                         "type": "network-error",
                         "url": "https://example.com/"
                        }`)
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://example.com/",
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

// The block of 'TestParseString_Example*' test specific examples from
// the current NEL doc:
// https://w3c.github.io/network-error-logging/#sample-network-error-reports
func TestParseString_Example3(t *testing.T) {
        msg := []byte(`
{
  "age": 0,
  "type": "network-error",
  "url": "https://www.example.com/",
  "body": {
    "sampling_fraction": 0.5,
    "referrer": "http://example.com/",
    "server_ip": "2001:DB8:0:0:0:0:0:42",
    "protocol": "h2",
    "method": "GET",
    "request_headers": {},
    "response_headers": {},
    "status_code": 200,
    "elapsed_time": 823,
    "phase": "application",
    "type": "http.protocol.error"
  }
}`)
        
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://www.example.com/",
                SamplingFraction: 0.5,
                Referrer: "http://example.com/",
                ServerIP: "2001:DB8:0:0:0:0:0:42",
                Protocol: "h2",
                Method: "GET",
                StatusCode: 200,
                ElapsedTime: 823,
                Phase: "application",
                BodyType: "http.protocol.error",
                RequestHeaders: map[string]any{},
                ResponseHeaders: map[string]any{},
                AdditionalBody: map[string]any{},
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

func TestParseString_Example4(t *testing.T) {
        msg := []byte(`
{
  "age": 0,
  "type": "network-error",
  "url": "https://widget.com/thing.js",
  "body": {
    "sampling_fraction": 1.0,
    "referrer": "https://www.example.com/",
    "server_ip": "",
    "protocol": "",
    "method": "GET",
    "request_headers": {},
    "response_headers": {},
    "status_code": 0,
    "elapsed_time": 143,
    "phase": "dns",
    "type": "dns.name_not_resolved"
  }
}`)
        
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://widget.com/thing.js",
                SamplingFraction: 1.0,
                Referrer: "https://www.example.com/",
                ServerIP: "",
                Protocol: "",
                Method: "GET",
                StatusCode: 0,
                ElapsedTime: 143,
                Phase: "dns",
                BodyType: "dns.name_not_resolved",
                RequestHeaders: map[string]any{},
                ResponseHeaders: map[string]any{},
                AdditionalBody: map[string]any{},
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

func TestParseString_Example6(t *testing.T) {
        msg := []byte(`
{
  "age": 0,
  "type": "network-error",
  "url": "https://new-subdomain.example.com/",
  "body": {
    "sampling_fraction": 1.0,
    "server_ip": "",
    "protocol": "http/1.1",
    "method": "GET",
    "request_headers": {},
    "response_headers": {},
    "status_code": 0,
    "elapsed_time": 48,
    "phase": "dns",
    "type": "dns.name_not_resolved"
  }
}`)
        
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://new-subdomain.example.com/",
                SamplingFraction: 1.0,
                ServerIP: "",
                Protocol: "http/1.1",
                Method: "GET",
                StatusCode: 0,
                ElapsedTime: 48,
                Phase: "dns",
                BodyType: "dns.name_not_resolved",
                RequestHeaders: map[string]any{},
                ResponseHeaders: map[string]any{},
                AdditionalBody: map[string]any{},
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

func TestParseString_Example8(t *testing.T) {
        msg := []byte(`
{
  "age": 0,
  "type": "network-error",
  "url": "https://example.com/",
  "body": {
    "sampling_fraction": 1.0,
    "server_ip": "192.0.2.1",
    "protocol": "http/1.1",
    "method": "GET",
    "request_headers": {},
    "response_headers": {
      "ETag": ["01234abcd"]
    },
    "status_code": 200,
    "elapsed_time": 1392,
    "phase": "application",
    "type": "ok"
  }
}`)
        
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://example.com/",
                SamplingFraction: 1.0,
                ServerIP: "192.0.2.1",
                Protocol: "http/1.1",
                Method: "GET",
                StatusCode: 200,
                ElapsedTime: 1392,
                Phase: "application",
                BodyType: "ok",
                RequestHeaders:  map[string]any{},
                ResponseHeaders: map[string]any{"ETag": []any{string("01234abcd")}},
                AdditionalBody: map[string]any{},
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

func TestParseString_Example9(t *testing.T) {
        msg := []byte(`
{
  "age": 0,
  "type": "network-error",
  "url": "https://example.com/",
  "body": {
    "sampling_fraction": 1.0,
    "server_ip": "192.0.2.1",
    "protocol": "http/1.1",
    "method": "GET",
    "request_headers": {
      "If-None-Match": ["01234abcd"]
    },
    "response_headers": {
      "ETag": ["01234abcd"]
    },
    "status_code": 304,
    "elapsed_time": 45,
    "phase": "application",
    "type": "ok"
  }
}`)
        
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://example.com/",
                SamplingFraction: 1.0,
                ServerIP: "192.0.2.1",
                Protocol: "http/1.1",
                Method: "GET",
                StatusCode: 304,
                ElapsedTime: 45,
                Phase: "application",
                BodyType: "ok",
                RequestHeaders:  map[string]any{"If-None-Match": []any{string("01234abcd")}},
                ResponseHeaders: map[string]any{"ETag": []any{string("01234abcd")}},
                AdditionalBody: map[string]any{},
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

func TestParseString_Example10(t *testing.T) {
        msg := []byte(`
{
  "age": 0,
  "type": "network-error",
  "url": "https://example.com/",
  "body": {
    "sampling_fraction": 1.0,
    "server_ip": "192.0.2.1",
    "protocol": "http/1.1",
    "method": "GET",
    "request_headers": {
      "If-None-Match": ["01234abcd"]
    },
    "response_headers": {
      "ETag": ["56789ef01"]
    },
    "status_code": 200,
    "elapsed_time": 935,
    "phase": "application",
    "type": "ok"
  }
}`)
        
        want := NelRecord{
                Age: 0,
                Type: "network-error",
                URL: "https://example.com/",
                SamplingFraction: 1.0,
                ServerIP: "192.0.2.1",
                Protocol: "http/1.1",
                Method: "GET",
                StatusCode: 200,
                ElapsedTime: 935,
                Phase: "application",
                BodyType: "ok",
                RequestHeaders:  map[string]any{"If-None-Match": []any{string("01234abcd")}},
                ResponseHeaders: map[string]any{"ETag": []any{string("56789ef01")}},
                AdditionalBody: map[string]any{},
        }
        
        n, err := ParseMessage(msg)
        if err != nil {
                t.Errorf("ParseMessage returned error: %v", err)
        }

        compareNelRecord(t, n, want)
}

