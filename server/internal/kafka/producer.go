package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"fifo-simulator/server/internal/models"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicMain = "fila-processamento"
	TopicDLQ  = "dead-letter-queue"
)

// Producer wraps Kafka writer for the main topic.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a Kafka writer for the main topic.
func NewProducer(brokers []string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        TopicMain,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
	}
	return &Producer{writer: w}
}

// PublishMessages sends job messages to the main topic. IntervalMs is delay between each message (0 = burst).
func (p *Producer) PublishMessages(ctx context.Context, jobs []models.JobMessage, intervalMs int) error {
	for i := range jobs {
		payload, err := json.Marshal(jobs[i])
		if err != nil {
			return err
		}
		err = p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(jobs[i].ID),
			Value: payload,
		})
		if err != nil {
			return err
		}
		if intervalMs > 0 && i < len(jobs)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}
	return nil
}

// PublishDLQ sends a message to the dead-letter queue.
func (p *Producer) PublishDLQ(ctx context.Context, msg models.JobMessage) error {
	writer := &kafka.Writer{
		Addr:     p.writer.Addr,
		Topic:    TopicDLQ,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.ID),
		Value: payload,
	})
}

// Close closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// EnsureTopics creates main and DLQ topics if not exist (using controller connection).
// Returns an error if Kafka is unreachable or topic creation fails.
func EnsureTopics(brokers []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers")
	}
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get controller: %w", err)
	}
	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("dial controller: %w", err)
	}
	defer controllerConn.Close()

	for _, topic := range []string{TopicMain, TopicDLQ} {
		err = controllerConn.CreateTopics(kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		if err != nil {
			// Topic may already exist (TopicExistsError); ignore and continue
			log.Printf("ensure topic %s: %v", topic, err)
		}
	}
	return nil
}
