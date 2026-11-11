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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// This ensures only the exact root URL loads the form
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Weather Monitor</title>
    <style>
        body { font-family: Arial, sans-serif; padding: 40px; background: #f4f4f9; }
        .container { max-width: 650px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        input, button { width: 100%; padding: 10px; margin: 10px 0; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box; }
        button { background: #007bff; color: white; cursor: pointer; border: none; font-weight: bold; font-size: 16px; }
        button:hover { background: #0056b3; }
        
        #result { margin-top: 25px; display: none; }
        .header-info { text-align: center; margin-bottom: 20px; border-bottom: 2px solid #f4f4f9; padding-bottom: 10px; }
        .header-info h3 { margin: 0 0 5px 0; color: #333; }
        .header-info p { margin: 0; color: #666; }
        
        .forecast-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 15px; }
        .forecast-card { background: #e9ecef; padding: 15px; border-radius: 8px; text-align: center; box-shadow: 0 2px 4px rgba(0,0,0,0.05); }
        .forecast-card .time { font-size: 0.85em; color: #555; margin-bottom: 10px; font-weight: bold; line-height: 1.4; }
        .forecast-card .temp { font-size: 1.6em; font-weight: bold; color: #ff5722; margin: 5px 0; }
        .forecast-card .wind { font-size: 0.85em; color: #17a2b8; font-weight: bold; }
        
        .error { color: #dc3545; text-align: center; font-weight: bold; padding: 10px; background: #f8d7da; border-radius: 4px; }
        .footer { margin-top: 30px; text-align: center; font-size: 0.85em; color: #666; border-top: 1px solid #eee; padding-top: 15px; line-height: 1.5; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Forecast Request</h2>
        <form id="weatherForm">
            <label>5-Digit ZIP Code:</label>
            <input type="text" id="zip" required pattern="[0-9]{5}" placeholder="e.g. 12601">
            
            <label>Forecast Days (1-5):</label>
            <input type="number" id="days" required min="1" max="5" value="2">
            
            <button type="submit">Submit Request</button>
        </form>
        
        <div id="result"></div>
        
        <div class="footer">
            <strong>Project2 for Cloud Computing Fall 2026</strong><br>
            R. O. Topaloglu, Ph.D., Adjunct Professor<br>
            Marist University
        </div>
    </div>

    <script>
        document.getElementById('weatherForm').addEventListener('submit', function(e) {
            e.preventDefault();
            
            const zip = document.getElementById('zip').value;
            const days = document.getElementById('days').value;
            const resultDiv = document.getElementById('result');
            
            resultDiv.style.display = 'block';
            resultDiv.innerHTML = '<p style="text-align:center; color:#666;">Fetching weather data...</p>';

            fetch('/weather?zip=' + zip + '&days=' + days)
                .then(response => {
                    if (!response.ok) throw new Error('Network response was not ok');
                    return response.json();
                })
                .then(data => {
                    // Build the header
                    let html = '<div class="header-info">';
                    html += '<h3>' + data.city + ' (' + data.zip_code + ')</h3>';
                    html += '<p>' + data.days + '-Day Forecast</p></div>';
                    
                    // Build the grid of cards
                    html += '<div class="forecast-grid">';
                    data.forecast.forEach(item => {
                        // Convert Unix timestamp to local time
                        const date = new Date(item.dt * 1000);
                        const dateStr = date.toLocaleDateString([], {weekday: 'short', month: 'short', day: 'numeric'});
                        const timeStr = date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
                        
                        html += '<div class="forecast-card">';
                        html += '<div class="time">' + dateStr + '<br>' + timeStr + '</div>';
                        html += '<div class="temp">' + Math.round(item.main.temp) + '&deg;F</div>';
                        html += '<div class="wind">Wind: ' + parseFloat(item.wind.speed).toFixed(1) + ' mph</div>';
                        html += '</div>';
                    });
                    html += '</div>';
                    
                    resultDiv.innerHTML = html;
                })
                .catch(error => {
                    resultDiv.innerHTML = '<div class="error">Error: Could not retrieve forecast. Please verify the ZIP code.</div>';
                });
        });
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
	
	// Record metrics for each weather API request.
	http.HandleFunc("/weather", metricsMiddleware(weatherHandler))

	// Expose metrics to Prometheus.
	http.Handle("/metrics", promhttp.Handler())

    log.Printf("Weather server running on port 8080 with %d workers",
        workerCount)

    log.Fatal(http.ListenAndServe(":8080", nil))
}
