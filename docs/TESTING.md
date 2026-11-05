# Testing Guide

Tests can be performed locally using Docker Desktop or on the Oracle Cloud deployment.

## 1. Verify the Application Is Running

From the project directory, check the Docker containers:

**Local:**

```bash
docker compose ps
```

**Oracle Cloud:**

```bash
sudo docker compose ps
```

Verify that all five services are running: `weather-app`, `kafka`, `weather-consumer`, `prometheus`, and `grafana`.

## 2. Test a Single Weather Request

The Weather API accepts two query parameters:

- `zip` — A five-digit US ZIP code
- `days` — Number of forecast days (1–5)

Open a browser and enter:

```text
http://localhost:8080/weather?zip=12601&days=2
```

Or run the following command in the terminal:

```bash
curl "http://localhost:8080/weather?zip=12601&days=2"
```

This requests a two-day forecast for Poughkeepsie, NY.

To test different locations or forecast durations, change the `zip` and `days` values.

**Examples:**

```text
http://localhost:8080/weather?zip=10001&days=3
http://localhost:8080/weather?zip=12601&days=5
http://localhost:8080/weather?zip=90210&days=1
```

A successful request returns JSON containing the city, ZIP code, number of forecast days, temperature, wind speed, and forecast timestamps.

**Note:** When testing directly on Oracle Cloud, run the `curl` command in the SSH terminal. The browser URLs above assume the API is accessible from the computer running the browser.

## 3. Test Concurrent Requests and Go Workers

The Weather API uses a pool of five Go workers to process requests concurrently.

To test this functionality, send five weather requests at approximately the same time.

### Windows PowerShell

Open a PowerShell terminal and run:

```powershell
$jobs = 1..5 | ForEach-Object {
    Start-Job -ScriptBlock {
        Invoke-RestMethod "http://localhost:8080/weather?zip=12601&days=2"
    }
}

$jobs | Wait-Job | Out-Null

$jobs | Receive-Job |
    Select-Object city, zip_code, days

$jobs | Remove-Job
```

### macOS / Linux / Oracle Cloud

Open a terminal and run:

```bash
for i in {1..5}; do
    curl -s -o /dev/null \
      -w "Request $i: HTTP %{http_code}\n" \
      "http://localhost:8080/weather?zip=12601&days=2" &
done

wait
```

All five requests should complete successfully with HTTP status `200`.

### Verify Worker Activity

To observe the Go workers processing requests, open another terminal.

**Local:**

```bash
docker compose logs -f weather-app
```

**Oracle Cloud:**

```bash
sudo docker compose logs -f weather-app
```

Example output:

```text
Worker 1 processing ZIP 12601 for 2 days
Worker 2 processing ZIP 12601 for 2 days
Worker 3 processing ZIP 12601 for 2 days
Worker 4 processing ZIP 12601 for 2 days
Worker 5 processing ZIP 12601 for 2 days
```

Worker numbers and execution order may vary.

This test verifies that the application can accept multiple simultaneous HTTP requests and distribute them across its Go worker pool.

## 4. Expected Results

| Test | Expected Result |
|---|---|
| Single Weather Request | JSON forecast returned successfully |
| Different ZIP Codes | Forecast corresponds to the requested location |
| Different Forecast Durations | Forecast corresponds to the requested number of days |
| Concurrent Requests | All five requests complete successfully |
| Worker Pool | Worker logs show requests being processed by Go workers |

The worker pool allows multiple weather requests to be processed concurrently rather than requiring every request to finish before the next one begins.