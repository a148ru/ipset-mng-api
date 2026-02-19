```bash
# Создаем необходимые директории
mkdir -p docker/mysql docker/clickhouse data

# Запускаем с Makefile
make build
make up

# Или вручную
docker compose up -d

# Проверяем логи
docker compose logs -f api

# Генерируем API ключ после запуска
make init-key
# или
docker compose exec api ./generate-key

# Останавливаем
make down
```