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

Логин и получение JWT токена:

```bash
curl -X POST http://localhost:
```