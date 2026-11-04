# Cloud Computing Project 2 — Weather Monitoring System

## Project Overview

This project CURRENTLY implements a weather monitoring system using Go, Docker, and the OpenWeatherMap API.

The application allows users to request weather forecasts by entering a ZIP code and the number of forecast days.

The system uses Go goroutines and channels to process multiple weather requests concurrently through a worker pool.

**Current Features:**
- Weather forecasts using OpenWeatherMap
- HTTP API accepting ZIP codes and forecast durations
- Support for 1-5 forecast days
- Concurrent processing using 5 Go workers
- Docker-based development for Windows and macOS
- JSON weather responses containing temperature and wind speed

## 1. Prerequisites

The following software is required:

- Docker Desktop
- Git
- Visual Studio Code (recommended)
- OpenWeatherMap API key

**Note:** Go does not need to be installed locally. The application is compiled and executed inside Docker using Go 1.25.

## 2. Clone the Repository

Open a terminal and run:

```bash
git clone https://github.com/jiervolino/CloudComputing-project2.git
cd CloudComputing-project2
```

If the repository is already cloned, navigate to the existing project directory instead.

## 3. Configure the OpenWeatherMap API Key

Create a `.env` file in the root directory.

Add the following line:

```env
OPENWEATHER_API_KEY=your_api_key_here
```

Replace `your_api_key_here` with your actual OpenWeatherMap API key.

The `.env` file is excluded from GitHub using `.gitignore` to protect the API key.

Each team member must create their own `.env` file.

## 4. Build and Run the Application

Make sure Docker Desktop is running.

From the project directory, execute:

```bash
docker compose up --build
```

This command builds the Go application inside Docker and starts the HTTP server.

Expected output:

```text
Weather server running on port 8080 with 5 workers
```

The application is now accessible at:

```text
http://localhost:8080
```

Keep this terminal running while testing the API.

## 5. Test a Single Weather Request

Open a web browser and enter:

```text
http://localhost:8080/weather?zip=12601&days=2
```

This requests a two-day forecast for ZIP code 12601 (Poughkeepsie, NY).

The application returns JSON containing:
- City name
- ZIP code
- Number of forecast days requested
- Temperature in Fahrenheit
- Wind speed in miles per hour
- Forecast timestamps

Example response structure:

```json
{
  "city": "Poughkeepsie",
  "days": 2,
  "forecast": [
    {
      "dt": 1793815200,
      "main": {
        "temp": 69.1
      },
      "wind": {
        "speed": 9.89
      }
    }
  ],
  "zip_code": "12601"
}
```

The forecast API provides data at three-hour intervals. A two-day request returns approximately 16 forecast entries.

## 6. Test Multiple Concurrent Requests — Windows

Keep Docker running in the first terminal.

Open a second PowerShell terminal in VS Code.

Run the following commands to send two weather requests concurrently:

```powershell
$job1 = Start-Job -ScriptBlock {
    Invoke-RestMethod "http://localhost:8080/weather?zip=12601&days=2"
}

$job2 = Start-Job -ScriptBlock {
    Invoke-RestMethod "http://localhost:8080/weather?zip=10001&days=3"
}

$job1, $job2 | Wait-Job | Out-Null

$job1, $job2 | Receive-Job |
    Select-Object city, zip_code, days

$job1, $job2 | Remove-Job
```

Expected output:

```text
city          zip_code days
----          -------- ----
Poughkeepsie  12601       2
New York      10001       3
```

The Docker terminal displays which worker processed each request.

Example:

```text
Worker 1 processing ZIP 12601 for 2 days
Worker 1 finished ZIP 12601
Worker 3 processing ZIP 10001 for 3 days
Worker 3 finished ZIP 10001
```

Worker numbers and execution order may vary.

## 7. Test Multiple Concurrent Requests — macOS

Keep Docker running in one terminal.

Open another terminal and execute:

```bash
curl -s "http://localhost:8080/weather?zip=12601&days=2" > /tmp/weather