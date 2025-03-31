package collector

import (
	"time"
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
