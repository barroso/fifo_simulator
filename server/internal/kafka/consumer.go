package kafka

import (
	"context"
	"encoding/json"
	"fifo-simulator/server/internal/metrics"
	"fifo-simulator/server/internal/models"
	"fifo-simulator/server/internal/processor"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// isCoordinatorOrTransientError returns true for errors that should be retried with backoff
// (e.g. Group Coordinator Not Available before broker is fully ready).
func isCoordinatorOrTransientError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Group Coordinator Not Available") ||
		strings.Contains(s, "coordinator") ||
		strings.Contains(s, "not available")
}

const consumerGroupID = "fifo-simulator-consumer"

// Consumer reads from the main topic and processes messages.
type Consumer struct {
	reader  *kafka.Reader
	proc    *processor.Processor
	metrics *metrics.Store
	onDone  func()
}

// NewConsumer creates a Kafka reader for the main topic and runs the process loop in a goroutine.
func NewConsumer(brokers []string, proc *processor.Processor, m *metrics.Store, onDone func()) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   TopicMain,
		GroupID: consumerGroupID,
	})
	return &Consumer{reader: r, proc: proc, metrics: m, onDone: onDone}
}

// Run starts consuming. It blocks until ctx is cancelled. Run in a goroutine.
func (c *Consumer) Run(ctx context.Context) {
	defer c.reader.Close()
	var backoff time.Duration
	var lastCoordinatorLog time.Time
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if isCoordinatorOrTransientError(err) {
				if time.Since(lastCoordinatorLog) > 15*time.Second {
					log.Printf("consumer waiting for group coordinator (retrying): %v", err)
					lastCoordinatorLog = time.Now()
				}
				backoff = coordinatorBackoff(backoff)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
			} else {
				log.Printf("consumer fetch error: %v", err)
				backoff = 0
			}
			continue
		}
		backoff = 0 // success, reset backoff
		var job models.JobMessage
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			log.Printf("consumer unmarshal error: %v", err)
			c.reader.CommitMessages(ctx, msg)
			continue
		}
		if c.metrics != nil {
			c.metrics.RecordInFlightStart()
		}
		c.proc.Process(ctx, job, func() {
			if c.metrics != nil {
				c.metrics.RecordInFlightEnd()
			}
			c.reader.CommitMessages(ctx, msg)
			if c.onDone != nil {
				c.onDone()
			}
		}, func() {
			if c.metrics != nil {
				c.metrics.RecordInFlightEnd()
			}
			c.reader.CommitMessages(ctx, msg)
		})
	}
}

// DLQConsumer reads from the DLQ and stores items for API display.
type DLQConsumer struct {
	reader *kafka.Reader
	store  DLQStore
}

// DLQStore is the interface for storing DLQ items (e.g. in-memory slice).
type DLQStore interface {
	Append(item models.DLQItem)
	All() []models.DLQItem
}

// NewDLQConsumer creates a reader for the DLQ topic.
func NewDLQConsumer(brokers []string, store DLQStore) *DLQConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   TopicDLQ,
		GroupID: "fifo-simulator-dlq",
	})
	return &DLQConsumer{reader: r, store: store}
}

const maxCoordinatorBackoff = 15 * time.Second

func coordinatorBackoff(prev time.Duration) time.Duration {
	if prev == 0 {
		return 2 * time.Second // first retry after 2s to give broker time
	}
	next := prev * 2
	if next > maxCoordinatorBackoff {
		next = maxCoordinatorBackoff
	}
	return next
}

// Run runs the DLQ consumer until ctx is cancelled.
func (c *DLQConsumer) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer c.reader.Close()
	var backoff time.Duration
	var lastCoordinatorLog time.Time
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if isCoordinatorOrTransientError(err) {
				if time.Since(lastCoordinatorLog) > 15*time.Second {
					log.Printf("dlq consumer waiting for group coordinator (retrying): %v", err)
					lastCoordinatorLog = time.Now()
				}
				backoff = coordinatorBackoff(backoff)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
			} else {
				log.Printf("dlq consumer fetch error: %v", err)
				backoff = 0
			}
			continue
		}
		backoff = 0 // success, reset backoff
		var job models.JobMessage
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			c.reader.CommitMessages(ctx, msg)
			continue
		}
		c.store.Append(models.DLQItem{
			ID:         job.ID,
			Type:       job.Type,
			CreatedAt:  job.CreatedAt,
			RetryCount: job.RetryCount,
			FailedAt:   msg.Time,
			Payload:    string(msg.Value),
		})
		c.reader.CommitMessages(ctx, msg)
	}
}
