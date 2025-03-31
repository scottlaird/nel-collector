package collector

import (
	"time"
)

// NelPostFormat describes the format of NEL reports on the wire from
// browsers.  See
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Network_Error_Logging
type NelPostFormat struct {
	Age  int64          `json:"age"`
	Type string         `json:"type"`
	URL  string         `json:"url"`
	Body map[string]any `json:"body"`
}

// NelRecord describes the semi-processed format of NEL reports that
// we want to use to insert into the DB.
type NelRecord struct {
	Timestamp time.Time
	Age       int64
	Type      string
	URL       string
	Hostname  string
	ClientIP  string // populated from X-Forwarded-For and/or the directly connected IP

	// These are all fields in `body` in the spec; I'm hoisting them into the main struct.
	SamplingFraction float64
	ElapsedTime      float64
	Phase            string
	BodyType         string // The top-level message and the body both have a `type` field, and they're semantically different and both usually provided.
	ServerIP         string
	Protocol         string
	Referrer         string // Note the correct spelling in NEL, unlike HTTP.
	Method           string
	RequestHeaders   map[string]any
	ResponseHeaders  map[string]any
	statusCodeFloat  float64
	StatusCode       int

	// This is really a JSON blob without any required structure.
	// It's whatever is left from the NelPostFormat's Body after
	// we've removed all of the known fields.
	AdditionalBody map[string]any
}
