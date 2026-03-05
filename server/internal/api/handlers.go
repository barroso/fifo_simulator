package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"
	"fifo-simulator/server/internal/dlq"
	"fifo-simulator/server/internal/kafka"
	"fifo-simulator/server/internal/metrics"
	"fifo-simulator/server/internal/models"
)

// Server holds dependencies for HTTP handlers.
type Server struct {
	Metrics      *metrics.Store
	DLQ          *dlq.Store
	Enqueue      func(jobs []models.JobMessage, intervalMs int) error
	KafkaBrokers []string

	// sseSubs holds one channel per connected SSE client; BroadcastToSSE sends to all.
	sseSubs   []chan struct{}
	sseSubsMu sync.Mutex
}

// JobConfigRequest is the body for POST /api/jobs.
type JobConfigRequest struct {
	Count       int    `json:"count"`
	JobType     string `json:"job_type"`
	DelayMs     int    `json:"delay_ms"`
	IntervalMs  int    `json:"interval_ms"`
	FailPercent int    `json:"fail_percent"`
}

// PostJobs creates jobs and enqueues them.
func (s *Server) PostJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req JobConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Count <= 0 || req.Count > 10000 {
		http.Error(w, "count must be 1-10000", http.StatusBadRequest)
		return
	}
	jt := models.JobType(req.JobType)
	if jt != models.JobTypeImage && jt != models.JobTypeEmail {
		jt = models.JobTypeEmail
	}
	if req.FailPercent < 0 {
		req.FailPercent = 0
	}
	if req.FailPercent > 100 {
		req.FailPercent = 100
	}

	jobs := make([]models.JobMessage, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		failSim := req.FailPercent > 0 && (i*100/req.Count) < req.FailPercent
		id := genID()
		jobs = append(jobs, models.JobMessage{
			ID:         id,
			Type:       jt,
			CreatedAt:  time.Now(),
			DelayMs:    req.DelayMs,
			RetryCount: 0,
			FailSim:    failSim,
		})
	}
	// Reset metrics for a fresh view of this test run
	s.Metrics.Reset()

	if len(s.KafkaBrokers) > 0 {
		_ = kafka.EnsureTopics(s.KafkaBrokers)
	}
	if err := s.Enqueue(jobs, req.IntervalMs); err != nil {
		http.Error(w, fmt.Sprintf("enqueue failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.Metrics.RecordPublished(int64(req.Count))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"count":  req.Count,
		"message": fmt.Sprintf("%d mensagens enviadas para a fila", req.Count),
	})
}

// GetMetrics returns current metrics snapshot.
func (s *Server) GetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Metrics.Snapshot())
}

// PostReset clears all in-memory metrics and DLQ entries (useful between test runs).
func (s *Server) PostReset(w http.ResponseWriter, r *http.Request) {
	s.Metrics.Reset()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// GetDLQ returns all DLQ items.
func (s *Server) GetDLQ(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.DLQ.All())
}

// GetHistory returns history for analysis dashboard.
func (s *Server) GetHistory(w http.ResponseWriter, r *http.Request) {
	queueSize, latency, processed := s.Metrics.History()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_size": queueSize,
		"latency":    latency,
		"processed":  processed,
	})
}

// GetEvents streams SSE events (metrics snapshots + heartbeat).
func (s *Server) GetEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	myCh := make(chan struct{}, 8)
	s.sseSubsMu.Lock()
	s.sseSubs = append(s.sseSubs, myCh)
	s.sseSubsMu.Unlock()
	defer func() {
		s.sseSubsMu.Lock()
		for i, c := range s.sseSubs {
			if c == myCh {
				s.sseSubs = append(s.sseSubs[:i], s.sseSubs[i+1:]...)
				break
			}
		}
		s.sseSubsMu.Unlock()
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	send := func() {
		snap := s.Metrics.Snapshot()
		data, _ := json.Marshal(snap)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			send()
		case <-myCh:
			send()
		}
	}
}

// BroadcastToSSE notifies all connected SSE clients that metrics changed (call from metrics.Notify).
func (s *Server) BroadcastToSSE() {
	s.sseSubsMu.Lock()
	subs := append([]chan struct{}(nil), s.sseSubs...)
	s.sseSubsMu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// CORS middleware.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func genID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// PostInternalEvent receives metric events from consumer containers and applies them to the local store.
// POST /internal/event — not exposed via nginx, only reachable inside the Docker network.
func (s *Server) PostInternalEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ev metrics.MetricEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	switch ev.Type {
	case "processed":
		s.Metrics.RecordProcessed(ev.LatencyMs)
	case "failed":
		s.Metrics.RecordFailed()
	case "dlq":
		s.Metrics.RecordDLQ()
	case "in_flight_start":
		s.Metrics.RecordInFlightStart()
	case "in_flight_end":
		s.Metrics.RecordInFlightEnd()
	default:
		http.Error(w, "unknown event type", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RunServer starts the HTTP server with the given handlers.
func RunServer(addr string, s *Server) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/jobs", s.PostJobs)
	mux.HandleFunc("/api/metrics", s.GetMetrics)
	mux.HandleFunc("/api/dlq", s.GetDLQ)
	mux.HandleFunc("/api/history", s.GetHistory)
	mux.HandleFunc("/api/events", s.GetEvents)
	mux.HandleFunc("/api/reset", s.PostReset)
	mux.HandleFunc("/internal/event", s.PostInternalEvent)
	return http.ListenAndServe(addr, CORS(mux))
}
