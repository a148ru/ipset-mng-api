package storage

import (
    "context"
    "fmt"
    "time"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    
    "github.com/ClickHouse/clickhouse-go/v2"
    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseKeyStorage - реализация для хранения ключей в ClickHouse
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
        SELECT 
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
            set_name String,
            ip String,
            cidr String,
            port UInt16,
            protocol String,
            description String,
            context String,
            set_type String,
            set_options String,
            created_at DateTime,
            updated_at DateTime,
            is_deleted UInt8 DEFAULT 0,
            version UInt32
        ) ENGINE = ReplacingMergeTree(version)
        ORDER BY (id, updated_at)
        SETTINGS index_granularity = 8192
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create ipset_records table: %v", err)
    }
    
    return &ClickHouseIPSetStorage{conn: conn}, nil
}

func (s *ClickHouseIPSetStorage) getNextID(ctx context.Context) (int, error) {
    // Получаем максимальный ID среди активных записей
    var maxID uint32
    err := s.conn.QueryRow(ctx, `
        SELECT MAX(id) 
        FROM ipset_records 
        WHERE is_deleted = 0
    `).Scan(&maxID)
    
    if err != nil {
        // Если ошибка из-за отсутствия данных, начинаем с 100000
        return 100000, nil
    }
    
    nextID := int(maxID) + 1
    if nextID < 100000 {
        nextID = 100000
    }
    if nextID > 999999 {
        return 0, fmt.Errorf("no available IDs in range 100000-999999")
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
    
    err = s.conn.Exec(ctx, `
        INSERT INTO ipset_records 
        (id, set_name, ip, cidr, port, protocol, description, context, set_type, set_options, created_at, updated_at, is_deleted, version)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        uint32(record.ID), record.SetName, record.IP, record.CIDR, uint16(record.Port), 
        record.Protocol, record.Description, record.Context, record.SetType, record.SetOptions,
        record.CreatedAt, record.UpdatedAt, uint8(0), uint32(1),
    )
    
    if err != nil {
        return fmt.Errorf("failed to create record: %v", err)
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) GetByID(id int) (*models.IPSetRecord, error) {
    ctx := context.Background()
    
    var record models.IPSetRecord
    // var isDeleted uint8
    //var version uint32

    //if isDeleted < 0 {}
    //if version < 0 {}
    
    err := s.conn.QueryRow(ctx, `
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE id = ? AND is_deleted = 0
        ORDER BY version DESC
        LIMIT 1
    `, uint32(id)).Scan(
        &record.ID, &record.SetName, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
        &record.Description, &record.Context, &record.SetType, &record.SetOptions,
        &record.CreatedAt, &record.UpdatedAt,
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
            id, set_name, ip, cidr, port, protocol, description, context, 
            set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE is_deleted = 0
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
            &record.ID, &record.SetName, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
            &record.Description, &record.Context, &record.SetType, &record.SetOptions,
            &record.CreatedAt, &record.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan record: %v", err)
        }
        records = append(records, &record)
    }
    
    return records, nil
}

func (s *ClickHouseIPSetStorage) GetBySetName(setName string) ([]*models.IPSetRecord, error) {
    ctx := context.Background()
    
    rows, err := s.conn.Query(ctx, `
        SELECT 
            id, set_name, ip, cidr, port, protocol, description, context, 
            set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE set_name = ? AND is_deleted = 0
        ORDER BY id
    `, setName)
    if err != nil {
        return nil, fmt.Errorf("failed to get records by set name: %v", err)
    }
    defer rows.Close()
    
    var records []*models.IPSetRecord
    for rows.Next() {
        var record models.IPSetRecord
        if err := rows.Scan(
            &record.ID, &record.SetName, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
            &record.Description, &record.Context, &record.SetType, &record.SetOptions,
            &record.CreatedAt, &record.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan record: %v", err)
        }
        records = append(records, &record)
    }
    
    if len(records) == 0 {
        return nil, fmt.Errorf("set %s not found", setName)
    }
    
    return records, nil
}

func (s *ClickHouseIPSetStorage) GetAllSets() ([]*models.IPSetSet, error) {
    ctx := context.Background()
    
    // Получаем уникальные сеты с агрегированной информацией
    rows, err := s.conn.Query(ctx, `
        SELECT 
            set_name,
            any(set_type) as set_type,
            any(set_options) as set_options,
            MIN(created_at) as created_at,
            MAX(updated_at) as updated_at,
            COUNT(*) as record_count
        FROM ipset_records
        WHERE is_deleted = 0
        GROUP BY set_name
        ORDER BY set_name
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to get all sets: %v", err)
    }
    defer rows.Close()
    
    var sets []*models.IPSetSet
    for rows.Next() {
        set := &models.IPSetSet{
            Records: []models.IPSetRecord{},
        }
        var recordCount uint64
        
        err := rows.Scan(&set.Name, &set.Type, &set.Options, &set.CreatedAt, &set.UpdatedAt, &recordCount)
        if err != nil {
            return nil, fmt.Errorf("failed to scan set: %v", err)
        }
        
        // Получаем записи для этого сета
        records, err := s.GetBySetName(set.Name)
        if err == nil {
            for _, r := range records {
                set.Records = append(set.Records, *r)
            }
        }
        
        sets = append(sets, set)
    }
    
    return sets, nil
}

func (s *ClickHouseIPSetStorage) Update(id int, record *models.IPSetRecord) error {
    ctx := context.Background()
    
    // Получаем текущую версию и created_at
    var currentVersion uint32
    var createdAt time.Time
    
    err := s.conn.QueryRow(ctx, `
        SELECT version, created_at
        FROM ipset_records
        WHERE id = ? AND is_deleted = 0
        ORDER BY version DESC
        LIMIT 1
    `, uint32(id)).Scan(&currentVersion, &createdAt)
    
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return fmt.Errorf("record with id %d not found", id)
        }
        return fmt.Errorf("failed to get current version: %v", err)
    }
    
    record.UpdatedAt = time.Now()
    record.CreatedAt = createdAt // Сохраняем оригинальную дату создания
    
    err = s.conn.Exec(ctx, `
        INSERT INTO ipset_records 
        (id, set_name, ip, cidr, port, protocol, description, context, set_type, set_options, created_at, updated_at, is_deleted, version)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        uint32(id), record.SetName, record.IP, record.CIDR, uint16(record.Port), 
        record.Protocol, record.Description, record.Context, record.SetType, record.SetOptions,
        record.CreatedAt, record.UpdatedAt, uint8(0), currentVersion+1,
    )
    
    if err != nil {
        return fmt.Errorf("failed to update record: %v", err)
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) Delete(id int) error {
    ctx := context.Background()
    
    // Получаем текущую версию и данные
    var currentVersion uint32
    var setName, ip, cidr, protocol, description, context, setType, setOptions string
    var port uint16
    var createdAt, updatedAt time.Time
    
    err := s.conn.QueryRow(ctx, `
        SELECT version, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE id = ? AND is_deleted = 0
        ORDER BY version DESC
        LIMIT 1
    `, uint32(id)).Scan(&currentVersion, &setName, &ip, &cidr, &port, &protocol, 
        &description, &context, &setType, &setOptions, &createdAt, &updatedAt)
    
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return fmt.Errorf("record with id %d not found", id)
        }
        return fmt.Errorf("failed to get record for deletion: %v", err)
    }
    
    // Вставляем запись с пометкой удаления
    err = s.conn.Exec(ctx, `
        INSERT INTO ipset_records 
        (id, set_name, ip, cidr, port, protocol, description, context, set_type, set_options, created_at, updated_at, is_deleted, version)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        uint32(id), setName, ip, cidr, port, protocol, description, context, 
        setType, setOptions, createdAt, time.Now(), uint8(1), currentVersion+1,
    )
    
    if err != nil {
        return fmt.Errorf("failed to delete record: %v", err)
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) DeleteSet(setName string) error {
    ctx := context.Background()
    
    // Получаем все записи сета
    rows, err := s.conn.Query(ctx, `
        SELECT id, version, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE set_name = ? AND is_deleted = 0
    `, setName)
    if err != nil {
        return fmt.Errorf("failed to get set records for deletion: %v", err)
    }
    defer rows.Close()
    
    var records []struct {
        id          uint32
        version     uint32
        setName     string
        ip          string
        cidr        string
        port        uint16
        protocol    string
        description string
        context     string
        setType     string
        setOptions  string
        createdAt   time.Time
        updatedAt   time.Time
    }
    
    for rows.Next() {
        var r struct {
            id          uint32
            version     uint32
            setName     string
            ip          string
            cidr        string
            port        uint16
            protocol    string
            description string
            context     string
            setType     string
            setOptions  string
            createdAt   time.Time
            updatedAt   time.Time
        }
        err := rows.Scan(&r.id, &r.version, &r.setName, &r.ip, &r.cidr, &r.port, 
            &r.protocol, &r.description, &r.context, &r.setType, &r.setOptions, 
            &r.createdAt, &r.updatedAt)
        if err != nil {
            return fmt.Errorf("failed to scan record: %v", err)
        }
        records = append(records, r)
    }
    
    if len(records) == 0 {
        return fmt.Errorf("set %s not found", setName)
    }
    
    // Помечаем все записи как удаленные
    for _, r := range records {
        err = s.conn.Exec(ctx, `
            INSERT INTO ipset_records 
            (id, set_name, ip, cidr, port, protocol, description, context, set_type, set_options, created_at, updated_at, is_deleted, version)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `,
            r.id, r.setName, r.ip, r.cidr, r.port, r.protocol, r.description, r.context,
            r.setType, r.setOptions, r.createdAt, time.Now(), uint8(1), r.version+1,
        )
        if err != nil {
            return fmt.Errorf("failed to delete record %d: %v", r.id, err)
        }
    }
    
    return nil
}

func (s *ClickHouseIPSetStorage) Search(query string) ([]*models.IPSetRecord, error) {
    ctx := context.Background()
    
    // ClickHouse поддерживает полнотекстовый поиск через токенизацию
    rows, err := s.conn.Query(ctx, `
        SELECT 
            id, set_name, ip, cidr, port, protocol, description, context, 
            set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE is_deleted = 0 
            AND (positionCaseInsensitive(context, ?) > 0 
                 OR positionCaseInsensitive(description, ?) > 0
                 OR positionCaseInsensitive(ip, ?) > 0
                 OR positionCaseInsensitive(set_name, ?) > 0)
        ORDER BY 
            CASE 
                WHEN positionCaseInsensitive(set_name, ?) = 1 THEN 1
                WHEN positionCaseInsensitive(ip, ?) = 1 THEN 2
                WHEN positionCaseInsensitive(context, ?) = 1 THEN 3
                WHEN positionCaseInsensitive(description, ?) = 1 THEN 4
                ELSE 5
            END,
            id
    `, query, query, query, query, query, query, query, query)
    
    if err != nil {
        return nil, fmt.Errorf("failed to search records: %v", err)
    }
    defer rows.Close()
    
    var records []*models.IPSetRecord
    for rows.Next() {
        var record models.IPSetRecord
        if err := rows.Scan(
            &record.ID, &record.SetName, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
            &record.Description, &record.Context, &record.SetType, &record.SetOptions,
            &record.CreatedAt, &record.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan record: %v", err)
        }
        records = append(records, &record)
    }
    
    return records, nil
}