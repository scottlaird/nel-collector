package collector

import (
	"encoding/json"
	"time"
)

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

func NewNELHandler(db DBConfig) *NELHandler {
	return &NELHandler{DB: db}
}
