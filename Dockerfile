FROM golang:1.25

WORKDIR /app

COPY go.mod ./
COPY . .

RUN go build -o /app/weather-app .

CMD ["/app/weather-app"]
