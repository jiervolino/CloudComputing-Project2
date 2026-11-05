# Cloud Computing Project 2 — Weather Monitoring System

## Overview

This project implements a cloud-based weather monitoring system using Go, Apache Kafka, Prometheus, Grafana, Docker, and the OpenWeatherMap API.

Users request weather forecasts through an HTTP API by specifying a ZIP code and the number of forecast days. A Go worker pool processes requests concurrently, publishes forecast data to Kafka, and returns JSON responses.

A separate Kafka consumer evaluates forecasts for extreme temperatures and wind conditions. Prometheus collects application metrics, which are visualized through Grafana.

The complete system is deployed on an Oracle Cloud Ubuntu virtual machine.

## Features

- HTTP weather API supporting ZIP codes and 1–5 forecast days
- Concurrent processing using five Go workers
- Kafka producer and consumer for weather events
- Alerts for temperatures above 90°F, below 32°F, and winds above 25 mph
- Prometheus metrics for requests, latency, and alerts
- Grafana monitoring dashboard
- Docker Compose deployment locally and on Oracle Cloud

## System Architecture

```text
User → Go Weather API → OpenWeatherMap
             |
             v
         Apache Kafka
             |
             v
       Weather Consumer (Alert Evaluation)


Weather API + Consumer
             |
             v
         Prometheus
             |
             v
           Grafana
```

## Prerequisites

- Docker Desktop
- Git
- OpenWeatherMap API key

Go does not need to be installed locally. The application is built inside Docker using Go 1.25.

## Quick Start

**1. Clone the repository**

```bash
git clone https://github.com/jiervolino/CloudComputing-project2.git
cd CloudComputing-project2
```

**2. Create a `.env` file**

```env
OPENWEATHER_API_KEY=your_api_key_here
```

**3. Start the application**

```bash
docker compose up --build -d
```

**4. Verify the containers**

```bash
docker compose ps
```

**5. Test the Weather API**

Open:

http://localhost:8080/weather?zip=12601&days=2

Or run:

```bash
curl "http://localhost:8080/weather?zip=12601&days=2"
```

A successful request returns JSON containing forecast temperatures, wind speeds, and timestamps.

## Monitoring

The application includes Prometheus and Grafana monitoring.

| Service | Local Address |
|---|---|
| Weather API | http://localhost:8080 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |

The Grafana dashboard visualizes request counts, request rates, latency, concurrent requests, and weather alerts.

## Documentation

- [Oracle Cloud Deployment](docs/DEPLOYMENT.md) — Cloud setup, Docker images, SSH access, and Grafana.
- [Testing Guide](docs/TESTING.md) — API testing, Kafka verification, concurrency, and performance measurements.

## Stop the Application

```bash
docker compose down
```

## References

- [Go](https://go.dev/)
- [Apache Kafka](https://kafka.apache.org/documentation/)
- [Docker](https://docs.docker.com/)
- [Prometheus](https://prometheus.io/docs/)
- [Grafana](https://grafana.com/docs/)
- [Oracle Cloud](https://docs.oracle.com/en-us/iaas/Content/home.htm)
- [OpenWeatherMap](https://openweathermap.org/api)