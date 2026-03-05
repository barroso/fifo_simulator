package metrics

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Reporter is implemented by Store (in-process) and HttpReporter (cross-process).
type Reporter interface {
	RecordProcessed(latencyMs int64)
	RecordFailed()
	RecordDLQ()
	RecordInFlightStart()
	RecordInFlightEnd()
}

// Compile-time check: Store must satisfy Reporter.
var _ Reporter = (*Store)(nil)

type metricEventType = string

const (
	eventProcessed     metricEventType = "processed"
	eventFailed        metricEventType = "failed"
	eventDLQ           metricEventType = "dlq"
	eventInFlightStart metricEventType = "in_flight_start"
	eventInFlightEnd   metricEventType = "in_flight_end"
)

// MetricEvent is the payload posted to /internal/event.
type MetricEvent struct {
	Type      metricEventType `json:"type"`
	LatencyMs int64           `json:"latency_ms,omitempty"`
}

// HttpReporter sends metric events to the API server over HTTP.
// Used by the consumer container to report back to the API.
type HttpReporter struct {
	apiURL     string
	httpClient *http.Client
}

func NewHttpReporter(apiURL string) *HttpReporter {
	return &HttpReporter{
		apiURL:     apiURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (r *HttpReporter) post(ev MetricEvent) {
	data, _ := json.Marshal(ev)
	req, err := http.NewRequest(http.MethodPost, r.apiURL+"/internal/event", bytes.NewReader(data))
	if err != nil {
		log.Printf("[metrics/http] build request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Printf("[metrics/http] post %s: %v", ev.Type, err)
		return
	}
	resp.Body.Close()
}

func (r *HttpReporter) RecordProcessed(latencyMs int64) {
	r.post(MetricEvent{Type: eventProcessed, LatencyMs: latencyMs})
}

func (r *HttpReporter) RecordFailed() {
	r.post(MetricEvent{Type: eventFailed})
}

func (r *HttpReporter) RecordDLQ() {
	r.post(MetricEvent{Type: eventDLQ})
}

func (r *HttpReporter) RecordInFlightStart() {
	r.post(MetricEvent{Type: eventInFlightStart})
}

func (r *HttpReporter) RecordInFlightEnd() {
	r.post(MetricEvent{Type: eventInFlightEnd})
}
