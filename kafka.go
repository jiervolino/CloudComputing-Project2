package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "time"

    "github.com/segmentio/kafka-go"
)

var kafkaWriter *kafka.Writer

// Initialize the Kafka producer.
func initKafka() {
    broker := os.Getenv("KAFKA_BROKER")
    if broker == "" {
        broker = "kafka:9092"
    }

    kafkaWriter = &kafka.Writer{
        Addr:         kafka.TCP(broker),
        Topic:        "weather-events",
        Balancer:     &kafka.Hash{},
        RequiredAcks: kafka.RequireAll,
        WriteTimeout: 5 * time.Second,
    }

    log.Println("Kafka producer initialized")
}

// Publish a weather forecast to Kafka.
func publishWeather(ctx context.Context, zip string, data interface{}) error {
    message, err := json.Marshal(data)
    if err != nil {
        return err
    }

    err = kafkaWriter.WriteMessages(ctx, kafka.Message{
        Key:   []byte(zip),
        Value: message,
    })

    if err != nil {
        return err
    }

    log.Printf("Published weather forecast for ZIP %s to Kafka", zip)
    return nil
}
