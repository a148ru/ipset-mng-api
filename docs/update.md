Новые API эндпоинты:
Управление ipset:
Создание ipset:

```bash
curl -X POST http://localhost:8080/ipsets \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "username_access",
    "type": "hash:net",
    "family": "inet",
    "hashsize": 1024,
    "maxelem": 65536,
    "description": "User access network"
  }'
```

Добавление записи в ipset:

```bash
curl -X POST http://localhost:8080/ipsets/username_access/entries \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "value": "192.168.0.0/16",
    "comment": "Internal network"
  }'
```

Создание ipset с портами:

```bash
curl -X POST http://localhost:8080/ipsets \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "dba_tools_access",
    "type": "hash:ip,port",
    "family": "inet",
    "hashsize": 1024,
    "maxelem": 65536,
    "description": "DBA tools access"
  }'

curl -X POST http://localhost:8080/ipsets/dba_tools_access/entries \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "value": "192.168.28.193,tcp:9644",
    "comment": "DBA tool 1"
  }'
```

Управление iptables правилами:

Создание правила iptables:

```bash
curl -X POST http://localhost:8080/iptables/rules \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "chain": "WG_ACCESS_F",
    "interface": "wg+",
    "src_sets": ["usergroup"],
    "dst_sets": ["dev_tools_access"],
    "action": "ACCEPT",
    "description": "Allow dev tools access",
    "position": 1
  }'
```

Применение конфигурации:

```bash
# Dry run (проверка без применения)
curl -X POST http://localhost:8080/apply?dry_run=true \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "ipset_commands": [
      {
        "command": "create",
        "set_name": "test_set",
        "args": ["hash:ip", "family", "inet", "hashsize", "1024", "maxelem", "65536"]
      }
    ],
    "iptables_commands": [
      "iptables -A INPUT -m set --match-set test_set src -j ACCEPT"
    ]
  }'

# Реальное применение
curl -X POST http://localhost:8080/apply \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "ipset_commands": [...],
    "iptables_commands": [...]
  }'
```

Импорт конфигурации из текста:

```bash
curl -X POST http://localhost:8080/import \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "config": "create username_access hash:net family inet hashsize 1024 maxelem 65536\nadd username_access 192.168.0.0/16\ncreate usergroup hash:ip family inet hashsize 1024 maxelem 65536\nadd usergroup 192.168.20.0/24"
  }'
```

Генерация команд:

```bash
# Сгенерировать команды для ipset
curl -X GET http://localhost:8080/generate/ipset/username_access \
  -H "Authorization: Bearer your-jwt-token"

# Сгенерировать команду для iptables правила
curl -X GET http://localhost:8080/generate/iptables/100001 \
  -H "Authorization: Bearer your-jwt-token"
```

Поиск:

```bash
# Поиск ipset
curl -X GET "http://localhost:8080/ipsets/search?q=access" \
  -H "Authorization: Bearer your-jwt-token"

# Поиск iptables правил
curl -X GET "http://localhost:8080/iptables/rules/search?q=WG_ACCESS" \
  -H "Authorization: Bearer your-jwt-token"
```