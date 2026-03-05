package processor

import (
	"context"
	"fifo-simulator/server/internal/metrics"
	"fifo-simulator/server/internal/models"
	"log"
	"math/rand"
	"time"
)

const maxRetries = 3

// ProducerDLQ is the interface for publishing to DLQ (implemented by kafka.Producer).
type ProducerDLQ interface {
	PublishDLQ(ctx context.Context, msg models.JobMessage) error
}

// ProducerRetry is the interface for re-publishing to main topic (for retry).
type ProducerRetry interface {
	PublishMessages(ctx context.Context, jobs []models.JobMessage, intervalMs int) error
}

// Processor processes a single job: delay, optional failure simulation, retry or DLQ.
type Processor struct {
	metrics *metrics.Store
	dlq     ProducerDLQ
	retry   ProducerRetry
}

// NewProcessor creates a processor with metrics and DLQ/retry publishers.
func NewProcessor(metricsStore *metrics.Store, dlq ProducerDLQ, retry ProducerRetry) *Processor {
	return &Processor{metrics: metricsStore, dlq: dlq, retry: retry}
}

// Process runs the job: delay by type, then succeed or fail (simulated). On failure, retry or send to DLQ.
// onSuccess and onRetry are called so the main consumer can commit the offset.
func (p *Processor) Process(ctx context.Context, job models.JobMessage, onSuccess, onRetry func()) {
	delay := p.delayForJob(job)
	log.Printf("[processor] job=%s type=%s failSim=%v retryCount=%d delay=%s",
		job.ID, job.Type, job.FailSim, job.RetryCount, delay.Round(time.Millisecond))

	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	// Simulate failure: FailSim is set when creating jobs (FailPercent in config).
	if job.FailSim {
		p.metrics.RecordFailed()
		job.RetryCount++
		if job.RetryCount >= maxRetries {
			log.Printf("[processor] job=%s FAILED max retries (%d) → DLQ", job.ID, maxRetries)
			if err := p.dlq.PublishDLQ(ctx, job); err != nil {
				log.Printf("[processor] dlq publish error for job=%s: %v", job.ID, err)
			}
			p.metrics.RecordDLQ()
			onSuccess() // commit so we don't reprocess
			return
		}
		log.Printf("[processor] job=%s FAILED (attempt %d/%d) → retry", job.ID, job.RetryCount, maxRetries)
		// Re-publish for retry (keep FailSim=true so it keeps simulating failure)
		if p.retry != nil {
			if err := p.retry.PublishMessages(ctx, []models.JobMessage{job}, 0); err != nil {
				log.Printf("[processor] retry publish error for job=%s: %v", job.ID, err)
			}
		}
		onRetry()
		return
	}

	// Success: record latency and processed
	latencyMs := time.Since(job.CreatedAt).Milliseconds()
	log.Printf("[processor] job=%s SUCCESS latency=%dms", job.ID, latencyMs)
	p.metrics.RecordProcessed(latencyMs)
	onSuccess()
}

func (p *Processor) delayForJob(job models.JobMessage) time.Duration {
	if job.DelayMs > 0 {
		return time.Duration(job.DelayMs) * time.Millisecond
	}
	switch job.Type {
	case models.JobTypeImage:
		return 200*time.Millisecond + time.Duration(rand.Intn(300))*time.Millisecond // 200–500ms
	case models.JobTypeEmail:
		return 20*time.Millisecond + time.Duration(rand.Intn(30))*time.Millisecond   // 20–50ms
	default:
		return 50 * time.Millisecond
	}
}

// PublishMessages is used by kafka.Producer to satisfy ProducerRetry; we need to avoid circular dependency.
// So the consumer will pass a small adapter that calls producer.PublishMessages.
var _ ProducerRetry = (*retryAdapter)(nil)

type retryAdapter struct {
	publish func(ctx context.Context, jobs []models.JobMessage, intervalMs int) error
}

func (r *retryAdapter) PublishMessages(ctx context.Context, jobs []models.JobMessage, intervalMs int) error {
	return r.publish(ctx, jobs, intervalMs)
}

// NewRetryAdapter creates a ProducerRetry from a function (e.g. producer.PublishMessages).
func NewRetryAdapter(publish func(ctx context.Context, jobs []models.JobMessage, intervalMs int) error) ProducerRetry {
	return &retryAdapter{publish: publish}
}
