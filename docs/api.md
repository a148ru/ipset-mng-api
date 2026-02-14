# API Документация

## Аутентификация

### Логин

```http
POST /login
Content-Type: application/json

{
    "api_key": "your-api-key"
}
```

Ответ:

```json
{
    "token": "jwt-token-here"
}
```

### Записи (Records)

#### Получить все записи

```http
GET /records
Authorization: Bearer <token>
```

#### Получить запись по ID

```http
GET /records/:id
Authorization: Bearer <token>
```

#### Создать запись

```http
POST /records
Authorization: Bearer <token>
Content-Type: application/json

{
    "set_name": "webservers",
    "ip": "192.168.1.100",
    "cidr": "32",
    "port": 80,
    "protocol": "tcp",
    "description": "Web server",
    "context": "production",
    "set_type": "hash:ip,port"
}
```

#### Обновить запись

```http
PUT /records/:id
Authorization: Bearer <token>
Content-Type: application/json

{
    "description": "Updated description"
}
```

#### Удалить запись

```http
DELETE /records/:id
Authorization: Bearer <token>
```
#### Поиск записей

```http
GET /records/search?q=query
Authorization: Bearer <token>
```

### Сеты (Sets)

#### Получить все сеты

```http
GET /sets
Authorization: Bearer <token>
```

#### Получить сет по имени

```http
GET /sets/:set_name
Authorization: Bearer <token>
```

#### Удалить сет

```http
DELETE /sets/:set_name
Authorization: Bearer <token>
```

#### Импортировать сет

```http
POST /sets/import
Authorization: Bearer <token>
Content-Type: application/json

{
    "set_name": "webservers",
    "set_type": "hash:ip,port",
    "records": [
        {"ip": "192.168.1.100", "port": 80, "protocol": "tcp"},
        {"ip": "192.168.1.101", "port": 443, "protocol": "tcp"}
    ],
    "context": "imported"
}
```

#### Экспортировать сет

```http
GET /sets/:set_name/export?format=ipset
Authorization: Bearer <token>
```