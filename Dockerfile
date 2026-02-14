FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ipset-api ./cmd/server/main.go
RUN go build -o generate-key ./cmd/generate_key/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/ipset-api .
COPY --from=builder /app/generate-key .
COPY .env.example .env

RUN mkdir -p data

EXPOSE 8080

CMD ["./ipset-api"]

