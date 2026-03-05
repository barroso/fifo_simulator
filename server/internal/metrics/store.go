package metrics

import (
	"sync"
	"time"
)

// Store holds in-memory metrics for the dashboard.
type Store struct {
	mu sync.RWMutex

	Published   int64   // total messages published (current test)
	Processed   int64   // successfully processed
	Failed      int64   // total failed attempts (retries count here too)
	DLQCount    int64   // sent to DLQ
	QueueSize   int64   // Published - Processed - DLQCount (pending/waiting)
	InFlight    int64   // messages actively being worked on by consumer right now
	LatencyMs   []int64 // recent success latencies for avg/p95
	MaxSamples  int
	StartedAt   time.Time
	LastEventAt time.Time

	// History for analysis dashboard (in-memory, last N points)
	QueueSizeHistory   []Point
	LatencyHistory     []Point
	ProcessedHistory   []Point
	MaxHistoryPoints   int

	// Notify is called (non-blocking) whenever any metric changes, for real-time SSE push.
	Notify func()
}

// Point is a timestamped value for charts.
type Point struct {
	Ts    int64   `json:"ts"`
	Value float64 `json:"value"`
}

// NewStore creates a metrics store with default caps.
func NewStore() *Store {
	return &Store{
		MaxSamples:      1000,
		MaxHistoryPoints: 500,
	}
}

func (s *Store) notify() {
	if s.Notify != nil {
		s.Notify()
	}
}

// RecordPublished increments published count and updates queue size.
func (s *Store) RecordPublished(n int64) {
	s.mu.Lock()
	if s.StartedAt.IsZero() {
		s.StartedAt = time.Now()
	}
	s.Published += n
	s.QueueSize += n
	s.LastEventAt = time.Now()
	s.appendQueueSizeHistory()
	s.mu.Unlock()
	s.notify()
}

// RecordInFlightStart marks a message as being actively processed.
func (s *Store) RecordInFlightStart() {
	s.mu.Lock()
	s.InFlight++
	s.mu.Unlock()
	s.notify()
}

// RecordInFlightEnd marks a message as done (success, retry committed, or DLQ).
func (s *Store) RecordInFlightEnd() {
	s.mu.Lock()
	s.InFlight--
	if s.InFlight < 0 {
		s.InFlight = 0
	}
	s.mu.Unlock()
}

// RecordProcessed increments processed, decrements queue size, records latency.
func (s *Store) RecordProcessed(latencyMs int64) {
	s.mu.Lock()
	s.Processed++
	s.QueueSize--
	if s.QueueSize < 0 {
		s.QueueSize = 0
	}
	s.LatencyMs = append(s.LatencyMs, latencyMs)
	if len(s.LatencyMs) > s.MaxSamples {
		s.LatencyMs = s.LatencyMs[len(s.LatencyMs)-s.MaxSamples:]
	}
	s.LastEventAt = time.Now()
	s.appendQueueSizeHistory()
	s.appendLatencyHistory(latencyMs)
	s.appendProcessedHistory()
	s.mu.Unlock()
	s.notify()
}

// RecordFailed increments failed count.
func (s *Store) RecordFailed() {
	s.mu.Lock()
	s.Failed++
	s.LastEventAt = time.Now()
	s.mu.Unlock()
	s.notify()
}

// RecordDLQ increments DLQ count and decrements queue size.
func (s *Store) RecordDLQ() {
	s.mu.Lock()
	s.DLQCount++
	s.QueueSize--
	if s.QueueSize < 0 {
		s.QueueSize = 0
	}
	s.LastEventAt = time.Now()
	s.appendQueueSizeHistory()
	s.mu.Unlock()
	s.notify()
}

func (s *Store) appendQueueSizeHistory() {
	ts := time.Now().UnixMilli()
	s.QueueSizeHistory = append(s.QueueSizeHistory, Point{Ts: ts, Value: float64(s.QueueSize)})
	if len(s.QueueSizeHistory) > s.MaxHistoryPoints {
		s.QueueSizeHistory = s.QueueSizeHistory[len(s.QueueSizeHistory)-s.MaxHistoryPoints:]
	}
}

func (s *Store) appendLatencyHistory(latencyMs int64) {
	ts := time.Now().UnixMilli()
	s.LatencyHistory = append(s.LatencyHistory, Point{Ts: ts, Value: float64(latencyMs)})
	if len(s.LatencyHistory) > s.MaxHistoryPoints {
		s.LatencyHistory = s.LatencyHistory[len(s.LatencyHistory)-s.MaxHistoryPoints:]
	}
}

func (s *Store) appendProcessedHistory() {
	ts := time.Now().UnixMilli()
	s.ProcessedHistory = append(s.ProcessedHistory, Point{Ts: ts, Value: float64(s.Processed)})
	if len(s.ProcessedHistory) > s.MaxHistoryPoints {
		s.ProcessedHistory = s.ProcessedHistory[len(s.ProcessedHistory)-s.MaxHistoryPoints:]
	}
}

// Snapshot returns a copy of current metrics for API/SSE.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	latencyAvg := int64(0)
	if len(s.LatencyMs) > 0 {
		var sum int64
		for _, v := range s.LatencyMs {
			sum += v
		}
		latencyAvg = sum / int64(len(s.LatencyMs))
	}
	latencyP95 := int64(0)
	if len(s.LatencyMs) > 0 {
		idx := int(float64(len(s.LatencyMs)) * 0.95)
		if idx >= len(s.LatencyMs) {
			idx = len(s.LatencyMs) - 1
		}
		latencyP95 = s.LatencyMs[idx]
	}

	finished := s.Processed + s.DLQCount
	var successRate float64
	if finished > 0 {
		successRate = float64(s.Processed) / float64(finished) * 100
	}

	var elapsedSec float64
	if !s.StartedAt.IsZero() {
		elapsedSec = time.Since(s.StartedAt).Seconds()
	}
	var throughput float64
	if elapsedSec > 1 {
		throughput = float64(s.Processed) / elapsedSec
	}

	return Snapshot{
		Published:     s.Published,
		Processed:     s.Processed,
		Failed:        s.Failed,
		DLQCount:      s.DLQCount,
		QueueSize:     s.QueueSize,
		InFlight:      s.InFlight,
		SuccessRate:   successRate,
		LatencyAvgMs:  latencyAvg,
		LatencyP95Ms:  latencyP95,
		ThroughputRps: throughput,
		LastEventAt:   s.LastEventAt,
	}
}

// Reset zeroes all counters and history (call between tests).
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Published = 0
	s.Processed = 0
	s.Failed = 0
	s.DLQCount = 0
	s.QueueSize = 0
	s.InFlight = 0
	s.LatencyMs = s.LatencyMs[:0]
	s.QueueSizeHistory = s.QueueSizeHistory[:0]
	s.LatencyHistory = s.LatencyHistory[:0]
	s.ProcessedHistory = s.ProcessedHistory[:0]
	s.StartedAt = time.Time{}
	s.LastEventAt = time.Time{}
}

// Snapshot is the JSON shape for /api/metrics and SSE.
type Snapshot struct {
	Published     int64     `json:"published"`
	Processed     int64     `json:"processed"`
	Failed        int64     `json:"failed"`
	DLQCount      int64     `json:"dlq_count"`
	QueueSize     int64     `json:"queue_size"`
	InFlight      int64     `json:"in_flight"`
	SuccessRate   float64   `json:"success_rate"`
	ThroughputRps float64   `json:"throughput_rps"`
	LatencyAvgMs  int64     `json:"latency_avg_ms"`
	LatencyP95Ms  int64     `json:"latency_p95_ms"`
	LastEventAt   time.Time `json:"last_event_at"`
}

// History returns copies of history slices for analysis API.
func (s *Store) History() (queueSize, latency, processed []Point) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	queueSize = append([]Point(nil), s.QueueSizeHistory...)
	latency = append([]Point(nil), s.LatencyHistory...)
	processed = append([]Point(nil), s.ProcessedHistory...)
	return queueSize, latency, processed
}
