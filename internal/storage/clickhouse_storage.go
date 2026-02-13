// internal/storage/clickhouse_storage.go
package storage

import (
    "context"
    "fmt"
    "time"
    "ipset-api/internal/config"
    "ipset-api/internal/models"
    
    "github.com/ClickHouse/clickhouse-go/v2"
    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouseKeyStorage struct {
    conn driver.Conn
}

func NewClickHouseKeyStorage(cfg *config.Config) (*ClickHouseKeyStorage, error) {
    ctx := context.Background()
    
    conn, err := clickhouse.Open(&clickhouse.Options{
        Addr: []string{fmt.Sprintf("%s:%s", cfg.ClickHouseHost, cfg.ClickHousePort)},
        Auth: clickhouse.Auth{
            Database: cfg.ClickHouseDatabase,
            Username: cfg.ClickHouseUsername,
            Password: cfg.ClickHousePassword,
        },
        Settings: clickhouse.Settings{
            "max_execution_time": 60,
        },
        DialTimeout:      time.Second * 30,
        MaxOpenConns:     10,
        MaxIdleConns:     5,
        ConnMaxLifetime:  time.Hour,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to connect to clickhouse: %v", err)
    }
    
    if err := conn.Ping(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping clickhouse: %v", err)
    }
    
    // Создаем базу данных если не существует
    err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.ClickHouseDatabase))
    if err != nil {
        return nil, fmt.Errorf("failed to create database: %v", err)
    }
    
    // Создаем таблицу для ключей
    err = conn.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS auth_keys (
            key String,
            created_at DateTime,
            expires_at DateTime,
            is_active UInt8,
            updated_at DateTime DEFAULT now()
        ) ENGINE = MergeTree()
        ORDER BY (key, created_at)
        SETTINGS index_granularity = 8192
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create auth_keys table: %v", err)
    }
    
    return &ClickHouseKeyStorage{conn: conn}, nil
}

func (s *ClickHouseKeyStorage) GetKey(key string) (*models.AuthKey, error) {
    ctx := context.Background()
    
    var authKey models.AuthKey
    var isActive uint8
    
    err := s.conn.QueryRow(ctx, `
        SELECT key, created_at, expires_at, is_active
        FROM auth_keys
        WHERE key = ?
        ORDER BY updated_at DESC
        LIMIT 1
    `, key).Scan(&authKey.Key, &authKey.CreatedAt, &authKey.ExpiresAt, &isActive)
    
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to get key: %v", err)
    }
    
    authKey.IsActive = isActive == 1
    return &authKey, nil
}

func (s *ClickHouseKeyStorage) SaveKey(key *models.AuthKey) error {
    ctx := context.Background()
    
    isActive := uint8(0)
    if key.IsActive {
        isActive = 1
    }
    
    err := s.conn.Exec(ctx, `
        INSERT INTO auth_keys (key, created_at, expires_at, is_active, updated_at)
        VALUES (?, ?, ?, ?, now())
    `, key.Key, key.CreatedAt, key.ExpiresAt, isActive)
    
    if err != nil {
        return fmt.Errorf("failed to save key: %v", err)
    }
    
    return nil
}

func (s *ClickHouseKeyStorage) DeleteKey(key string) error {
    // ClickHouse не поддерживает DELETE напрямую, поэтому используем INSERT с пометкой удаления
    // или создаем таблицу-мутацию. В данном случае просто игнорируем или можно создать 
    // отдельную таблицу для удаленных записей
    ctx := context.Background()
    
    // Помечаем ключ как неактивный вместо удаления
    err := s.conn.Exec(ctx, `
        INSERT INTO auth_keys (key, created_at, expires_at, is_active, updated_at)
        SELECT key, created_at, expires_at, 0, now()
        FROM auth_keys
        WHERE key = ?
        ORDER BY updated_at DESC
        LIMIT 1
    `, key)
    
    if err != nil {
        return fmt.Errorf("failed to deactivate key: %v", err)
    }
    
    return nil
}

func (s *ClickHouseKeyStorage) ListKeys() ([]*models.AuthKey, error) {
    ctx := context.Background()
    
    rows, err := s.conn.Query(ctx, `
        SELECT DISTINCT ON (key) 
            key, created_at, expires_at, is_active
        FROM auth_keys
        ORDER BY key, updated_at DESC
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to list keys: %v", err)
    }
    defer rows.Close()
    
    var keys []*models.AuthKey
    for rows.Next() {
        var key models.AuthKey
        var isActive uint8
        
        if err := rows.Scan(&key.Key, &key.CreatedAt, &key.ExpiresAt, &isActive); err != nil {
            return nil, fmt.Errorf("failed to scan key: %v", err)
        }
        
        key.IsActive = isActive == 1
        keys = append(keys, &key)
    }
    
    return keys, nil
}

// ClickHouseIPSetStorage
type ClickHouseIPSetStorage struct {
    conn driver.Conn
}

func NewClickHouseIPSetStorage(cfg *config.Config) (*ClickHouseIPSetStorage, error) {
    ctx := context.Background()
    
    conn, err := clickhouse.Open(&clickhouse.Options{
        Addr: []string{fmt.Sprintf("%s:%s", cfg.ClickHouseHost, cfg.ClickHousePort)},
        Auth: clickhouse.Auth{
            Database: cfg.ClickHouseDatabase,
            Username: cfg.ClickHouseUsername,
            Password: cfg.ClickHousePassword,
        },
        Settings: clickhouse.Settings{
            "max_execution_time": 60,
        },
        DialTimeout:     time.Second * 30,
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: time.Hour,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to connect to clickhouse: %v", err)
    }
    
    if err := conn.Ping(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping clickhouse: %v", err)
    }
    
    // Создаем таблицу для ipset записей
    err = conn.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS ipset_records (
            id UInt32,
            ip String,
            cidr String,
            port UInt16,
            protocol String,
            description String,
            context String,
            created_at DateTime,
            updated_at DateTime,
            is_deleted UInt8 DEFAULT 0,
            version UInt32 DEFAULT 1
        ) ENGINE = ReplacingMergeTree(version)
        ORDER BY (id, updated_at)
        SETTINGS index_granularity = 8192
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create ipset_records table: %v", err)
    }
    
    // Создаем материализованное представление для полнотекстового поиска
    err = conn.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS ipset_records_search (
            id UInt32,
            context String,
            description String,
            tokens Array(String)
        ) ENGINE = MergeTree()
        ORDER BY id
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create search table: %v", err)
    }
    
    return &ClickHouseIPSetStorage{conn: conn}, nil
}

func (s *ClickHouseIPSetStorage) getNextID(ctx context.Context) (int, error) {
    // Получаем максимальный ID
    var maxID uint32
    err := s.conn.QueryRow(ctx, `
        SELECT MAX(id) 
        FROM ipset_records 
        WHERE is_deleted = 0
    `).Scan(&maxID)
    
    if err != nil {
        // Если ошибка из-за отсутствия данных, начинаем с 100000
        if err.Error() == "sql: no rows in result set" {
            return 100000, nil
        }
        return 0, fmt.Errorf("failed to get max ID: %v", err)
    }
    
    nextID := int(maxID) + 1
    if nextID < 100000 {
        nextID = 100000
    }
    if nextID > 999999 {
        // Ищем свободный ID, если достигли максимума
        var freeID uint32
        err = s.conn.QueryRow(ctx, `
            SELECT number
            FROM numbers(100000, 900000)
            WHERE number NOT IN (SELECT id FROM ipset_records WHERE is_deleted = 0)
            LIMIT 1
        `).Scan(&freeID)
        
        if err != nil {
            return 0, fmt.Errorf("no available IDs: %v", err)
        }
        nextID = int(freeID)
    }
    
    return nextID, nil
}

func (s *ClickHouseIPSetStorage) Create(record *models.IPSetRecord) error {
    ctx := context.Background()
    
    // Получаем следующий ID
    id, err := s.getNextID(ctx)
    if err != nil {
        return err
    }
    
    record.ID = id
    now := time.Now()
    record.CreatedAt = now
    record.UpdatedAt = now
    
    batch, err := s.conn.PrepareBatch(ctx, `
        INSERT INTO ipset_records (id, ip, cidr, port, protocol, description, context, created_at, updated_at, is_deleted, version)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare batch: %v", err)
    }
    
    err = batch.Append(
        uint32(record.ID), record.IP, record.CIDR, uint16(record.Port), 
        record.Protocol, record.Description, record.Context, 
        record.CreatedAt, record.UpdatedAt, uint8(0), uint32(1),
    )
    if err != nil {
        return fmt.Errorf("failed to append to batch: %v", err)
    }
    
    if err := batch.Send(); err != nil {
        return fmt.Errorf("failed to send batch: %v", err)
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) GetByID(id int) (*models.IPSetRecord, error) {
    ctx := context.Background()
    
    var record models.IPSetRecord
    var isDeleted uint8
    var version uint32
    
    err := s.conn.QueryRow(ctx, `
        SELECT id, ip, cidr, port, protocol, description, context, created_at, updated_at, is_deleted
        FROM ipset_records
        WHERE id = ? AND is_deleted = 0
        ORDER BY version DESC
        LIMIT 1
    `, uint32(id)).Scan(
        &record.ID, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
        &record.Description, &record.Context, &record.CreatedAt, &record.UpdatedAt,
        &isDeleted,
    )
    
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return nil, fmt.Errorf("record with id %d not found", id)
        }
        return nil, fmt.Errorf("failed to get record: %v", err)
    }
    
    return &record, nil
}

func (s *ClickHouseIPSetStorage) GetAll() ([]*models.IPSetRecord, error) {
    ctx := context.Background()
    
    rows, err := s.conn.Query(ctx, `
        SELECT 
            id, ip, cidr, port, protocol, description, context, 
            created_at, updated_at
        FROM (
            SELECT *,
                   ROW_NUMBER() OVER (PARTITION BY id ORDER BY version DESC) AS rn
            FROM ipset_records
            WHERE is_deleted = 0
        )
        WHERE rn = 1
        ORDER BY id
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to get all records: %v", err)
    }
    defer rows.Close()
    
    var records []*models.IPSetRecord
    for rows.Next() {
        var record models.IPSetRecord
        if err := rows.Scan(
            &record.ID, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
            &record.Description, &record.Context, &record.CreatedAt, &record.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan record: %v", err)
        }
        records = append(records, &record)
    }
    
    return records, nil
}

func (s *ClickHouseIPSetStorage) Update(id int, record *models.IPSetRecord) error {
    ctx := context.Background()
    
    // Получаем текущую версию
    var currentVersion uint32
    err := s.conn.QueryRow(ctx, `
        SELECT version
        FROM ipset_records
        WHERE id = ? AND is_deleted = 0
        ORDER BY version DESC
        LIMIT 1
    `, uint32(id)).Scan(&currentVersion)
    
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return fmt.Errorf("record with id %d not found", id)
        }
        return fmt.Errorf("failed to get current version: %v", err)
    }
    
    record.UpdatedAt = time.Now()
    
    batch, err := s.conn.PrepareBatch(ctx, `
        INSERT INTO ipset_records (id, ip, cidr, port, protocol, description, context, created_at, updated_at, is_deleted, version)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare batch: %v", err)
    }
    
    err = batch.Append(
        uint32(id), record.IP, record.CIDR, uint16(record.Port), 
        record.Protocol, record.Description, record.Context, 
        record.CreatedAt, record.UpdatedAt, uint8(0), currentVersion+1,
    )
    if err != nil {
        return fmt.Errorf("failed to append to batch: %v", err)
    }
    
    if err := batch.Send(); err != nil {
        return fmt.Errorf("failed to send batch: %v", err)
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) Delete(id int) error {
    ctx := context.Background()
    
    // Получаем текущую версию и данные
    var currentVersion uint32
    var ip, cidr, protocol, description, context string
    var port uint16
    var createdAt time.Time
    
    err := s.conn.QueryRow(ctx, `
        SELECT version, ip, cidr, port, protocol, description, context, created_at
        FROM ipset_records
        WHERE id = ? AND is_deleted = 0
        ORDER BY version DESC
        LIMIT 1
    `, uint32(id)).Scan(&currentVersion, &ip, &cidr, &port, &protocol, &description, &context, &createdAt)
    
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return fmt.Errorf("record with id %d not found", id)
        }
        return fmt.Errorf("failed to get record for deletion: %v", err)
    }
    
    // Вставляем запись с пометкой удаления
    batch, err := s.conn.PrepareBatch(ctx, `
        INSERT INTO ipset_records (id, ip, cidr, port, protocol, description, context, created_at, updated_at, is_deleted, version)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare batch: %v", err)
    }
    
    err = batch.Append(
        uint32(id), ip, cidr, port, protocol, description, context, 
        createdAt, time.Now(), uint8(1), currentVersion+1,
    )
    if err != nil {
        return fmt.Errorf("failed to append to batch: %v", err)
    }
    
    if err := batch.Send(); err != nil {
        return fmt.Errorf("failed to send batch: %v", err)
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) Search(context string) ([]*models.IPSetRecord, error) {
    ctx := context.Background()
    
    // ClickHouse поддерживает полнотекстовый поиск через токенизацию
    query := `
        SELECT 
            id, ip, cidr, port, protocol, description, context, 
            created_at, updated_at
        FROM (
            SELECT *,
                   ROW_NUMBER() OVER (PARTITION BY id ORDER BY version DESC) AS rn
            FROM ipset_records
            WHERE is_deleted = 0
        )
        WHERE rn = 1 
            AND (positionCaseInsensitive(context, ?) > 0 
                 OR positionCaseInsensitive(description, ?) > 0)
        ORDER BY 
            multiIf(
                positionCaseInsensitive(context, ?) = 1, 1,
                positionCaseInsensitive(description, ?) = 1, 2,
                positionCaseInsensitive(context, ?) > 0, 3,
                positionCaseInsensitive(description, ?) > 0, 4,
                5
            ),
            id
    `
    
    rows, err := s.conn.Query(ctx, query, context, context, context, context, context, context)
    if err != nil {
        return nil, fmt.Errorf("failed to search records: %v", err)
    }
    defer rows.Close()
    
    var records []*models.IPSetRecord
    for rows.Next() {
        var record models.IPSetRecord
        if err := rows.Scan(
            &record.ID, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
            &record.Description, &record.Context, &record.CreatedAt, &record.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan record: %v", err)
        }
        records = append(records, &record)
    }
    
    return records, nil
}