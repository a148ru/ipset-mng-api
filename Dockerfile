#######################

# Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ipset-api ./cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o generate-key ./cmd/generate_key/main.go

FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates tzdata ipset iptables bash curl

WORKDIR /app

# Копируем бинарные файлы
COPY --from=builder /app/ipset-api .
COPY --from=builder /app/generate-key .

# Создаем директорию для данных
RUN mkdir -p /app/data

# Скрипт ожидания готовности БД
COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./ipset-api"]