# Oracle Cloud Deployment Guide

This document explains how the Weather Monitoring System is deployed and managed on Oracle Cloud Infrastructure (OCI).

The application consists of five Docker services: Weather API, Apache Kafka, Weather Consumer, Prometheus, and Grafana.

## 1. Cloud Environment

| Component | Configuration |
|---|---|
| Cloud Provider | Oracle Cloud Infrastructure |
| Operating System | Ubuntu 22.04.5 LTS |
| VM Shape | VM.Standard.E2.1.Micro |
| Architecture | AMD64 |
| CPU | 1 OCPU |
| RAM | Approximately 1 GB |
| Additional Swap | 2 GB |
| Container Platform | Docker Engine and Docker Compose |

Due to the VM's limited memory, the Go application images were built locally and transferred to Oracle Cloud instead of being compiled on the VM.

## 2. Connect to Oracle Cloud

Open Windows PowerShell and run:

```powershell
ssh -i "PATH_TO_PRIVATE_SSH_KEY" ubuntu@132.145.166.241
```

Replace `PATH_TO_PRIVATE_SSH_KEY` with the path to your SSH private key. The IP address is specific to the existing project VM and must be updated if it changes.

Once connected, navigate to the project:

```bash
cd ~/CloudComputing-project2
```

## 3. Check and Start the Application

Check the status of all containers:

```bash
sudo docker compose ps
```

The following five services should be running:

- `weather-app`
- `kafka`
- `weather-consumer`
- `prometheus`
- `grafana`

If the containers are already running, no further startup steps are needed.

If they are stopped, start them using the existing Docker images:

```bash
sudo docker compose up -d --no-build
```


View application logs:

```bash
sudo docker compose logs --tail=30
```

Stop the application when necessary:

```bash
sudo docker compose down
```

Do not use `down -v` unless you intentionally want to delete the named volumes and their stored data.

## 4. Build and Transfer Docker Images

The Go images are built on a local computer to avoid exhausting the cloud VM's limited memory.

The commands below are run from Windows PowerShell in the project directory.

### Build for AMD64

```powershell
docker build --platform linux/amd64 -t weather-app:oracle .
```

For the consumer, use its Dockerfile. The following assumes `consumer/Dockerfile`:

```powershell
docker build --platform linux/amd64 -f consumer/Dockerfile -t weather-consumer:oracle .
```

The consumer Dockerfile path and build context should match the actual repository layout.

### Export the Images

```powershell
docker save -o weather-app-oracle.tar weather-app:oracle
docker save -o weather-consumer-oracle.tar weather-consumer:oracle
```

### Transfer to Oracle

```powershell
scp -i "PATH_TO_PRIVATE_SSH_KEY" .\weather-app-oracle.tar ubuntu@132.145.166.241:~/

scp -i "PATH_TO_PRIVATE_SSH_KEY" .\weather-consumer-oracle.tar ubuntu@132.145.166.241:~/
```

### Load Images on Oracle

Reconnect to the VM and run:

```bash
sudo docker load -i ~/weather-app-oracle.tar
sudo docker load -i ~/weather-consumer-oracle.tar
```

Verify:

```bash
sudo docker images
```

Both `weather-app:oracle` and `weather-consumer:oracle` should appear.

## 5. Access Grafana from Windows

Grafana runs on port 3000 of the Oracle VM and is bound to localhost.

Open a **second PowerShell terminal** and run:

```powershell
ssh -i "PATH_TO_PRIVATE_SSH_KEY" -L 3300:localhost:3000 ubuntu@132.145.166.241
```

Keep this terminal open.

In a browser, visit:

http://localhost:3300

Log in and navigate to:

**Dashboards → Oracle Cloud Project**

The Grafana Prometheus data source uses:

```text
http://prometheus:9090
```

The SSH tunnel allows access to Grafana without exposing its administrative interface publicly.

## 6. Useful Maintenance Commands

| Command | Purpose |
|---|---|
| `sudo docker compose ps` | Check service status |
| `sudo docker compose up -d --no-build` | Start existing deployment |
| `sudo docker compose logs --tail=30` | View recent logs |
| `sudo docker compose logs -f weather-app` | Follow API logs |
| `sudo docker compose logs -f weather-consumer` | Follow consumer logs |
| `sudo docker stats --no-stream` | View container resource usage |
| `free -h` | Check VM memory and swap |
| `sudo docker compose down` | Stop containers |