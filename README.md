### Инструкция по использованию:

#### Установка зависимостей:

```bash
go mod tidy
```

#### Настройка окружения:

```bash
cp .env.example .env
# Отредактируйте .env файл под ваши нужды
```

#### Генерация API ключа:

```bash
go run cmd/generate_key/main.go
```

#### Запуск сервера:

```bash
go run main.go
```

#### Использование API:

- Логин и получение JWT токена:

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"api_key":"your-generated-api-key"}'
```

- Создание записи:

```bash
curl -X POST http://localhost:8080/records \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "ip": "192.168.1.1",
    "cidr": "32",
    "port": 80,
    "protocol": "tcp",
    "description": "Web server",
    "context": "production web server"
  }'
```

- Получение всех записей:

```bash
curl -X GET http://localhost:8080/records \
  -H "Authorization: Bearer your-jwt-token"
```

- Получение записи по ID:

```bash
curl -X GET http://localhost:8080/records/100001 \
  -H "Authorization: Bearer your-jwt-token"
```

- Поиск записей:

```bash
curl -X GET "http://localhost:8080/records/search?q=web" \
  -H "Authorization: Bearer your-jwt-token"
```

- Обновление записи:

```bash
curl -X PUT http://localhost:8080/records/100001 \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated web server"}'
```

- Удаление записи:

```bash
curl -X DELETE http://localhost:8080/records/100001 \
  -H "Authorization: Bearer your-jwt-token"
```