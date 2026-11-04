FROM golang:1.25

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY consumer/ ./consumer/

RUN go build -o /app/weather-consumer ./consumer

CMD ["/app/weather-consumer"]