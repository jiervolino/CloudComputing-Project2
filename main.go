package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
    workerCount = 5
    queueSize   = 100
)

var client = &http.Client{
    Timeout: 10 * time.Second,
}

// Each job represents one user's weather request.
type WeatherJob struct {
    ZipCode string
    Days    int
    Ctx     context.Context
    Result  chan WeatherResult
}

// Each worker sends its result through this channel.
type WeatherResult struct {
    Data interface{}
    Err  error
}

// Shared channel containing pending weather requests.
var jobs = make(chan WeatherJob, queueSize)

// Retrieves real weather data from OpenWeatherMap.
func fetchWeather(ctx context.Context, zip string, days int) (interface{}, error) {
    apiKey := os.Getenv("OPENWEATHER_API_KEY")
    if apiKey == "" {
        return nil, errors.New("weather API key is not configured")
    }

    params := url.Values{}
    params.Set("zip", zip+",US")
    params.Set("appid", apiKey)
    params.Set("units", "imperial")

    apiURL := "https://api.openweathermap.org/data/2.5/forecast?" + params.Encode()

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("OpenWeatherMap returned status %d", resp.StatusCode)
    }

    var forecast struct {
        City struct {
            Name string `json:"name"`
        } `json:"city"`

        List []struct {
            Timestamp int64 `json:"dt"`

            Main struct {
                Temp float64 `json:"temp"`
            } `json:"main"`

            Wind struct {
                Speed float64 `json:"speed"`
            } `json:"wind"`
        } `json:"list"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&forecast); err != nil {
        return nil, err
    }

    count := days * 8
    if count > len(forecast.List) {
        count = len(forecast.List)
    }

    return map[string]interface{}{
        "zip_code": zip,
        "city":     forecast.City.Name,
        "days":     days,
        "forecast": forecast.List[:count],
    }, nil
}

// A Weather worker waits for jobs and processes them.
func weatherWorker(id int, jobs <-chan WeatherJob) {
    for job := range jobs {
        if job.Ctx.Err() != nil {
            continue
        }

        log.Printf("Worker %d processing ZIP %s for %d days",
            id, job.ZipCode, job.Days)

        data, err := fetchWeather(job.Ctx, job.ZipCode, job.Days)

		// Publish to Kafka if fetch was successful.
		if err == nil {
			if publishErr := publishWeather(job.Ctx, job.ZipCode, data); publishErr != nil {
				log.Printf("Kafka publish failed for ZIP %s: %v",
					job.ZipCode, publishErr)
				err = publishErr
			}
		}

        // Do not block if the user has disconnected.
        select {
        case job.Result <- WeatherResult{
            Data: data,
            Err:  err,
        }:
        case <-job.Ctx.Done():
        }

        log.Printf("Worker %d finished ZIP %s", id, job.ZipCode)
    }
}

// HTTP handler for incoming weather requests.
func weatherHandler(w http.ResponseWriter, r *http.Request) {
    zip := strings.TrimSpace(r.URL.Query().Get("zip"))
    daysStr := r.URL.Query().Get("days")

    days, err := strconv.Atoi(daysStr)

    if len(zip) != 5 || err != nil || days < 1 || days > 5 {
        http.Error(w, "Provide a 5-digit ZIP code and days between 1 and 5",
            http.StatusBadRequest)
        return
    }

    for _, ch := range zip {
        if ch < '0' || ch > '9' {
            http.Error(w, "ZIP code must contain only digits",
                http.StatusBadRequest)
            return
        }
    }

    // Each request receives its own result channel.
    resultChan := make(chan WeatherResult, 1)

    job := WeatherJob{
        ZipCode: zip,
        Days:    days,
        Ctx:     r.Context(),
        Result:  resultChan,
    }

    // Add the request to the shared jobs channel.
    select {
    case jobs <- job:
    case <-r.Context().Done():
        return
    }

    // Wait for this request's result.
    select {
    case result := <-resultChan:
        if result.Err != nil {
            log.Printf("Weather request failed: %v", result.Err)
            http.Error(w, "Unable to retrieve weather forecast",
                http.StatusBadGateway)
            return
        }

        w.Header().Set("Content-Type", "application/json")

        if err := json.NewEncoder(w).Encode(result.Data); err != nil {
            log.Printf("Failed to write response: %v", err)
        }

    case <-r.Context().Done():
        return
    }
}

func main() {

	// Initialize Kafka producer
	initKafka()
	defer kafkaWriter.Close()

    // Start five worker goroutines.
    for i := 1; i <= workerCount; i++ {
        go weatherWorker(i, jobs)
    }

	// Record metrics for each weather API request.
	http.HandleFunc("/weather", metricsMiddleware(weatherHandler))

	// Expose metrics to Prometheus.
	http.Handle("/metrics", promhttp.Handler())

    log.Printf("Weather server running on port 8080 with %d workers",
        workerCount)

    log.Fatal(http.ListenAndServe(":8080", nil))
}
