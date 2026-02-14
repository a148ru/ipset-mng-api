# IPSet API Manager

Управление IPSet правилами через REST API с поддержкой различных хранилищ.

## Возможности

- 🔐 JWT аутентификация
- 📦 CRUD операции для IPSet записей
- 🗄 Поддержка различных хранилищ (файл, MySQL, PostgreSQL, ClickHouse)
- 🔍 Поиск по контексту
- 📤 Импорт из существующих ipset файлов
- 📥 Экспорт в ipset формат
- 🐳 Docker поддержка
- 🖥 Удобный CLI интерфейс

## Быстрый старт

### Запуск сервера

```bash
# Клонировать репозиторий
git clone https://github.com/yourusername/ipset-api.git
cd ipset-api

# Скопировать конфигурацию
cp .env.example .env

# Сгенерировать API ключ
go run cmd/generate_key/main.go

# Запустить с Docker
docker-compose up -d

# Или локально
go run cmd/server/main.go
```

# Установка CLI

```bash
# Собрать CLI
cd cmd/cli
go build -o ipset-cli
sudo mv ipset-cli /usr/local/bin/

# Настроить
ipset-cli config set api_url http://localhost:8080
ipset-cli login your-api-key-here
```
