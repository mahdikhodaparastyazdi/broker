FROM golang:1.19-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /broker-service ./cmd/main.go

FROM scratch

COPY --from=builder /broker-service /broker-service

EXPOSE 8080

ENTRYPOINT ["/broker-service"]
