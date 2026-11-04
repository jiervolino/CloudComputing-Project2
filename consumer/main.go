package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "strconv"
    "time"

    "github.com/segmentio/kafka-go"
)

// Matches the JSON produced by the weather API.
type ForecastEntry struct {
    Dt int64 `json:"dt"`
    Main struct {
        Temp float64 `json:"temp"`
    } `json:"main"`
    Wind struct {
        Speed float64 `json:"speed"`
    } `json:"wind"`
}

type WeatherEvent struct {
    City     string          `json:"city"`
    ZipCode  string          `json:"zip_code"`
    Days     int             `json:"days"`
    Forecast []ForecastEntry `json:"forecast"`
}

func getThreshold(name string, defaultValue float64) float64 {
    value := os.Getenv(name)
    if value == "" {
        return defaultValue
    }

    parsed, err := strconv.ParseFloat(value, 64)
    if err != nil {
        log.Printf("Invalid %s=%q; using default %.1f",
            name, value, defaultValue)
        return defaultValue
    }

    return parsed
}

func evaluateWeather(event WeatherEvent, highTemp, lowTemp, highWind float64) {
    log.Printf("Evaluating %d forecast entries for %s (%s)",
        len(event.Forecast), event.City, event.ZipCode)

    alertCount := 0

    for _, entry := range event.Forecast {
        forecastTime := time.Unix(entry.Dt, 0).UTC().Format(time.RFC3339)

        if entry.Main.Temp > highTemp {
            log.Printf("HEAT ALERT | ZIP=%s | Time=%s | Temp=%.2f°F | Threshold=%.2f°F",
                event.ZipCode, forecastTime, entry.Main.Temp, highTemp)
            alertCount++
        }

        if entry.Main.Temp < lowTemp {
            log.Printf("FREEZING ALERT | ZIP=%s | Time=%s | Temp=%.2f°F | Threshold=%.2f°F",
                event.ZipCode, forecastTime, entry.Main.Temp, lowTemp)
            alertCount++
        }

        if entry.Wind.Speed > highWind {
            log.Printf("WIND ALERT | ZIP=%s | Time=%s | Wind=%.2f mph | Threshold=%.2f mph",
                event.ZipCode, forecastTime, entry.Wind.Speed, highWind)
            alertCount++
        }
    }

    if alertCount == 0 {
        log.Printf("No weather alerts for %s (%s)",
            event.City, event.ZipCode)
    } else {
        log.Printf("Generated %d weather alerts for ZIP %s",
            alertCount, event.ZipCode)
    }
}

func main() {
    broker := os.Getenv("KAFKA_BROKER")
    if broker == "" {
        broker = "kafka:9092"
    }

    highTemp := getThreshold("HIGH_TEMP_THRESHOLD", 90)
    lowTemp := getThreshold("LOW_TEMP_THRESHOLD", 32)
    highWind := getThreshold("HIGH_WIND_THRESHOLD", 25)

    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{broker},
        Topic:    "weather-events",
        GroupID:  "weather-alert-evaluator",
        MinBytes: 1,
        MaxBytes: 10e6,
    })
    defer reader.Close()

    log.Printf("Weather alert consumer started; broker=%s", broker)
    log.Printf("Thresholds: high=%.1f°F, low=%.1f°F, wind=%.1f mph",
        highTemp, lowTemp, highWind)

    for {
        // FetchMessage allows us to commit only after processing.
        message, err := reader.FetchMessage(context.Background())
        if err != nil {
            log.Printf("Kafka read error: %v", err)
            time.Sleep(time.Second)
            continue
        }

        var event WeatherEvent
        if err := json.Unmarshal(message.Value, &event); err != nil {
            log.Printf("Invalid weather event JSON: %v", err)
            // Leave the offset uncommitted and stop rather
            // than silently discard an invalid message.
            return
        }

        evaluateWeather(event, highTemp, lowTemp, highWind)

        if err := reader.CommitMessages(context.Background(), message); err != nil {
            log.Printf("Kafka commit error: %v", err)
        }
    }
}
