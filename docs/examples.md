# Примеры использования

## 1. Базовое управление

### Создание набора правил для веб-серверов

```bash
# Создаем сет для веб-серверов
ipset-cli records create \
  --set-name webservers \
  --ip 192.168.1.10 \
  --port 80 \
  --protocol tcp \
  --description "Web server 1" \
  --context "production"

ipset-cli records create \
  --set-name webservers \
  --ip 192.168.1.11 \
  --port 443 \
  --protocol tcp \
  --description "Web server 2" \
  --context "production"

# Просмотр созданного сета
ipset-cli sets get webservers

# Экспорт правил для iptables
ipset-cli sets export webservers > webservers.rules
```

## 2. Импорт существующих правил

### Импорт из /etc/ipset

Допустим, у вас есть файл `/etc/ipset`:

```text
create webservers hash:ip,port
add webservers 192.168.1.10,tcp:80
add webservers 192.168.1.11,tcp:443
create blacklist hash:ip
add blacklist 1.1.1.1
add blacklist 2.2.2.2
```

```bash
# Импорт с префиксом "legacy"
ipset-cli import etc --context-prefix legacy

# Результат:
# Importing set: webservers (2 rules)
#   ✅ 192.168.1.10:80
#   ✅ 192.168.1.11:443
# Importing set: blacklist (2 rules)
#   ✅ 1.1.1.1
#   ✅ 2.2.2.2
```

## 3. Миграция между серверами

### Экспорт с одного сервера и импорт на другой
```bash
# На старом сервере
ipset-cli --output ipset export > backup.rules

# Копируем файл на новый сервер
scp backup.rules new-server:~/

# На новом сервере
ipset-cli import file backup.rules --context-prefix migrated
```

### Прямая передача через SSH

```bash
ssh old-server "ipset save" | ipset-cli import stdin --context-prefix live
```

## 4. Интеграция с iptables

### Создание и применение правил

```bash
# Создаем правила через API
ipset-cli records create \
  --set-name allowed_ips \
  --ip 192.168.1.0 \
  --cidr 24 \
  --context "internal network"

ipset-cli records create \
  --set-name allowed_ips \
  --ip 10.0.0.0 \
  --cidr 8 \
  --context "vpn network"

# Экспортируем в ipset
ipset-cli sets export allowed_ips > allowed_ips.rules

# Применяем правила
bash allowed_ips.rules

# Настраиваем iptables
iptables -A INPUT -m set --match-set allowed_ips src -j ACCEPT
iptables -A INPUT -j DROP
```

## 5. Автоматизация с cron

### Ежечасный бэкап

```bash
#!/bin/bash
# /usr/local/bin/backup-ipset.sh

BACKUP_DIR="/var/backups/ipset"
DATE=$(date +%Y%m%d-%H%M%S)

mkdir -p $BACKUP_DIR
ipset-cli --output json export > "$BACKUP_DIR/ipset-backup-$DATE.json"
ipset-cli --output ipset export > "$BACKUP_DIR/ipset-backup-$DATE.rules"

# Оставляем только последние 24 бэкапа
ls -t $BACKUP_DIR/ipset-backup-*.json | tail -n +25 | xargs -r rm
```

##### В cron:

```cron
0 * * * * /usr/local/bin/backup-ipset.sh
6. Сложные запросы с jq
```

#### Анализ данных
```bash
# Получить все IP из сета webservers
ipset-cli --output json sets get webservers | jq -r '.records[].ip'

# Статистика по типам сетов
ipset-cli --output json sets list | jq 'group_by(.type) | map({type: .[0].type, count: length})'

# Поиск записей с определенным портом
ipset-cli --output json records list | jq '.[] | select(.port == 80)'
```

## 7. Интерактивный режим с fzf

```bash
#!/bin/bash
# ipset-fzf.sh

select_record() {
    ipset-cli --output json records list | \
        jq -r '.[] | "\(.id): \(.set_name) \(.ip) - \(.description)"' | \
        fzf --prompt="Select record: "
}

record=$(select_record)
if [ ! -z "$record" ]; then
    id=$(echo $record | cut -d':' -f1)
    ipset-cli records get $id
fi
```

## 8. Мониторинг и уведомления

### Проверка изменений

```bash
#!/bin/bash
# /usr/local/bin/check-ipset-changes.sh

CURRENT_HASH=$(ipset-cli --output json export | md5sum)
LAST_HASH=$(cat /var/run/ipset-last-hash 2>/dev/null)

if [ "$CURRENT_HASH" != "$LAST_HASH" ]; then
    echo "IPSet rules have changed!" | mail -s "IPSet Change Alert" admin@example.com
    echo $CURRENT_HASH > /var/run/ipset-last-hash
fi
```

## 9. Работа с разными окружениями

### Использование нескольких профилей

```bash
# Создаем разные конфиги
cp ~/.ipset-cli.yaml ~/.ipset-cli-prod.yaml
cp ~/.ipset-cli.yaml ~/.ipset-cli-dev.yaml

# Редактируем для разных окружений
# prod: api_url: https://api.prod.example.com
# dev:  api_url: https://api.dev.example.com

# Используем с разными окружениями
alias ipset-prod='ipset-cli --config ~/.ipset-cli-prod.yaml'
alias ipset-dev='ipset-cli --config ~/.ipset-cli-dev.yaml'

ipset-prod records list
ipset-dev records list
```

## 10. Полный цикл управления

```bash
#!/bin/bash
# Пример полного цикла управления

# 1. Создаем сет
ipset-cli records create \
  --set-name mail_servers \
  --ip 192.168.1.20 \
  --port 25 \
  --protocol tcp \
  --description "SMTP server" \
  --context "production"

# 2. Добавляем еще серверов
for ip in 192.168.1.21 192.168.1.22; do
    ipset-cli records create \
      --set-name mail_servers \
      --ip $ip \
      --port 25 \
      --protocol tcp \
      --description "SMTP server" \
      --context "production"
done

# 3. Проверяем созданное
ipset-cli sets get mail_servers

# 4. Экспортируем для iptables
ipset-cli sets export mail_servers > /etc/ipset.d/mail_servers.conf

# 5. Применяем
bash /etc/ipset.d/mail_servers.conf

# 6. Настраиваем iptables
iptables -A INPUT -p tcp --dport 25 -m set --match-set mail_servers src -j ACCEPT
iptables -A INPUT -p tcp --dport 25 -j DROP

# 7. Делаем бэкап
ipset-cli --output json export > "backup-$(date +%Y%m%d).json"

# 8. При необходимости удаляем
# ipset-cli sets delete mail_servers
```
