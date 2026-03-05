package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"fifo-simulator/server/internal/kafka"
	"fifo-simulator/server/internal/metrics"
	"fifo-simulator/server/internal/processor"
)

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	brokerList := strings.Split(brokers, ",")
	apiURL := getEnv("API_URL", "http://server:8080")
	consumerID := getEnv("CONSUMER_ID", "consumer-1")

	consumerDelay := 3 * time.Second
	if d := os.Getenv("KAFKA_CONSUMER_START_DELAY"); d != "" {
		if v, err := time.ParseDuration(d); err == nil && v >= 0 {
			consumerDelay = v
		}
	}

	log.Printf("[%s] starting — brokers=%s api=%s delay=%s", consumerID, brokers, apiURL, consumerDelay)
	time.Sleep(consumerDelay)

	reporter := metrics.NewHttpReporter(apiURL)

	prod := kafka.NewProducer(brokerList)
	defer prod.Close()

	retryAdapter := processor.NewRetryAdapter(prod.PublishMessages)
	proc := processor.NewProcessor(reporter, prod, retryAdapter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := kafka.NewConsumer(brokerList, proc, reporter, nil)
	go consumer.Run(ctx)

	log.Printf("[%s] consuming from fila-processamento", consumerID)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("[%s] shutting down", consumerID)
	cancel()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
