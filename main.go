package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"
)

var client = &http.Client{
    Timeout: 10 * time.Second,
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    zip := strings.TrimSpace(r.URL.Query().Get("zip"))
    daysStr := r.URL.Query().Get("days")

    days, err := strconv.Atoi(daysStr)

    if len(zip) != 5 || daysStr == "" || err != nil || days < 1 || days > 5 {
        http.Error(w, "Provide a 5-digit ZIP code and days between 1 and 5", http.StatusBadRequest)
        return
    }

    for _, ch := range zip {
        if ch < '0' || ch > '9' {
            http.Error(w, "ZIP code must contain only digits", http.StatusBadRequest)
            return
        }
    }

    apiKey := os.Getenv("OPENWEATHER_API_KEY")
    if apiKey == "" {
        http.Error(w, "Weather API is not configured", http.StatusInternalServerError)
        return
    }

    params := url.Values{}
    params.Set("zip", zip+",US")
    params.Set("appid", apiKey)
    params.Set("units", "imperial")

    apiURL := "https://api.openweathermap.org/data/2.5/forecast?" + params.Encode()

    req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, apiURL, nil)
    if err != nil {
        http.Error(w, "Unable to create weather request", http.StatusInternalServerError)
        return
    }

    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, "Unable to contact OpenWeatherMap", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("OpenWeatherMap returned status %d", resp.StatusCode)
        http.Error(w, fmt.Sprintf("Weather service returned status %d", resp.StatusCode), http.StatusBadGateway)
        return
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
        http.Error(w, "Unable to read weather data", http.StatusBadGateway)
        return
    }

    // The API returns one forecast approximately every 3 hours.
    // Eight entries represent approximately one day.
    count := days * 8
    if count > len(forecast.List) {
        count = len(forecast.List)
    }

    result := map[string]interface{}{
        "zip_code": zip,
        "city":     forecast.City.Name,
        "days":     days,
        "forecast": forecast.List[:count],
    }

    if err := json.NewEncoder(w).Encode(result); err != nil {
        log.Printf("Unable to send response: %v", err)
    }
}

func main() {
    http.HandleFunc("/weather", weatherHandler)

    fmt.Println("Weather server running on port 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
