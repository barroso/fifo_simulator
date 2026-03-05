package models

import "time"

// JobType represents the type of simulated task (affects delay).
type JobType string

const (
	JobTypeImage JobType = "image"
	JobTypeEmail JobType = "email"
)

// JobMessage is the payload sent to Kafka.
type JobMessage struct {
	ID        string    `json:"id"`
	Type      JobType   `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	DelayMs   int       `json:"delay_ms,omitempty"`
	RetryCount int      `json:"retry_count,omitempty"`
	FailSim   bool      `json:"fail_sim,omitempty"` // simulate failure for demo
}

// JobConfig is the request body for POST /api/jobs.
type JobConfig struct {
	Count        int     `json:"count"`          // number of records to enqueue
	JobType      JobType `json:"job_type"`       // "image" | "email"
	DelayMs      int     `json:"delay_ms"`       // base delay in ms (0 = use default by type)
	IntervalMs   int     `json:"interval_ms"`     // ms between each message publish (0 = burst)
	FailPercent  int     `json:"fail_percent"`    // 0-100, percent of jobs that will "fail" and retry
}

// DLQItem is an item stored in the dead letter queue view.
type DLQItem struct {
	ID         string    `json:"id"`
	Type       JobType   `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	RetryCount int       `json:"retry_count"`
	FailedAt   time.Time `json:"failed_at"`
	Payload    string    `json:"payload,omitempty"`
}
