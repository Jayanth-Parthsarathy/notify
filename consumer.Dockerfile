FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o consumer cmd/consumer/consumer.go

CMD ["./consumer"]
