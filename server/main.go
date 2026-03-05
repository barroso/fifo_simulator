package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"fifo-simulator/server/internal/api"
	"fifo-simulator/server/internal/dlq"
	"fifo-simulator/server/internal/kafka"
	"fifo-simulator/server/internal/metrics"
	"fifo-simulator/server/internal/models"
)

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	brokerList := strings.Split(brokers, ",")
	httpAddr := getEnv("HTTP_ADDR", ":8080")

	// Wait for Kafka and create topics (retry until ready)
	for i := 0; i < 30; i++ {
		if err := kafka.EnsureTopics(brokerList); err == nil {
			log.Printf("kafka topics ready")
			break
		}
		if i == 29 {
			log.Printf("warn: could not create kafka topics after 30 attempts, will retry on first request")
		} else {
			time.Sleep(2 * time.Second)
		}
	}

	metricsStore := metrics.NewStore()
	dlqStore := dlq.NewStore(500)

	prod := kafka.NewProducer(brokerList)
	defer prod.Close()

	// DLQ consumer stays in the API server since it feeds the /api/dlq endpoint.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dlqConsumer := kafka.NewDLQConsumer(brokerList, dlqStore)
	var wg sync.WaitGroup
	wg.Add(1)
	go dlqConsumer.Run(ctx, &wg)

	enqueue := func(jobs []models.JobMessage, intervalMs int) error {
		return prod.PublishMessages(context.Background(), jobs, intervalMs)
	}

	srv := &api.Server{
		Metrics:      metricsStore,
		DLQ:          dlqStore,
		Enqueue:      enqueue,
		KafkaBrokers: brokerList,
	}
	metricsStore.Notify = func() { srv.BroadcastToSSE() }

	go func() {
		if err := api.RunServer(httpAddr, srv); err != nil {
			log.Printf("http server error: %v", err)
		}
	}()

	log.Printf("api server listening on %s", httpAddr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	wg.Wait()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
