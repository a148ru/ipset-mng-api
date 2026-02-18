#!/bin/bash
# docker-entrypoint.sh

set -e

echo "Waiting for database to be ready..."

# Функция для проверки MySQL
wait_for_mysql() {
    echo "Testing MySQL connection to $MYSQL_HOST:$MYSQL_PORT as $MYSQL_USERNAME"
    for i in {1..30}; do
        if mysqladmin ping -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USERNAME" -p"$MYSQL_PASSWORD" --silent; then
            echo "MySQL is ready!"
            return 0
        fi
        echo "Waiting for MySQL... ($i/30)"
        sleep 2
    done
    echo "MySQL is not ready after 30 attempts"
    return 1
}

# Функция для проверки PostgreSQL
wait_for_postgres() {
    echo "Testing PostgreSQL connection to $POSTGRES_HOST:$POSTGRES_PORT"
    for i in {1..30}; do
        if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USERNAME" -d "$POSTGRES_DATABASE"; then
            echo "PostgreSQL is ready!"
            return 0
        fi
        echo "Waiting for PostgreSQL... ($i/30)"
        sleep 2
    done
    echo "PostgreSQL is not ready after 30 attempts"
    return 1
}

# Ждем нужную БД в зависимости от типа хранилища
if [ "$AUTH_STORAGE_TYPE" = "mysql" ] || [ "$IPSET_STORAGE_TYPE" = "mysql" ]; then
    wait_for_mysql || exit 1
fi

if [ "$AUTH_STORAGE_TYPE" = "postgresql" ] || [ "$IPSET_STORAGE_TYPE" = "postgresql" ]; then
    wait_for_postgres || exit 1
fi

# Запускаем приложение
exec "$@"
