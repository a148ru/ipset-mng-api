package storage

import (
    "database/sql"
    "fmt"
    "time"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    
    _ "github.com/lib/pq"
)

// PostgreSQLKeyStorage - реализация для хранения ключей в PostgreSQL
type PostgreSQLKeyStorage struct {
    db *sql.DB
}

func NewPostgreSQLKeyStorage(cfg *config.Config) (*PostgreSQLKeyStorage, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.PostgreSQLHost,
        cfg.PostgreSQLPort,
        cfg.PostgreSQLUsername,
        cfg.PostgreSQLPassword,
        cfg.PostgreSQLDatabase,
    )
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to postgresql: %v", err)
    }
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping postgresql: %v", err)
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS auth_keys (
            key VARCHAR(255) PRIMARY KEY,
            created_at TIMESTAMP WITH TIME ZONE,
            expires_at TIMESTAMP WITH TIME ZONE,
            is_active BOOLEAN DEFAULT true
        )
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create auth_keys table: %v", err)
    }
    
    return &PostgreSQLKeyStorage{db: db}, nil
}


func (s *PostgreSQLKeyStorage) GetKey(key string) (*models.AuthKey, error) {
    var authKey models.AuthKey
    err := s.db.QueryRow(
        "SELECT key, created_at, expires_at, is_active FROM auth_keys WHERE key = $1",
        key,
    ).Scan(&authKey.Key, &authKey.CreatedAt, &authKey.ExpiresAt, &authKey.IsActive)
    
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get key: %v", err)
    }
    
    return &authKey, nil
}

func (s *PostgreSQLKeyStorage) SaveKey(key *models.AuthKey) error {
    _, err := s.db.Exec(
        `INSERT INTO auth_keys (key, created_at, expires_at, is_active) 
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (key) DO UPDATE 
         SET created_at = $2, expires_at = $3, is_active = $4`,
        key.Key, key.CreatedAt, key.ExpiresAt, key.IsActive,
    )
    if err != nil {
        return fmt.Errorf("failed to save key: %v", err)
    }
    return nil
}

func (s *PostgreSQLKeyStorage) DeleteKey(key string) error {
    _, err := s.db.Exec("DELETE FROM auth_keys WHERE key = $1", key)
    if err != nil {
        return fmt.Errorf("failed to delete key: %v", err)
    }
    return nil
}

func (s *PostgreSQLKeyStorage) ListKeys() ([]*models.AuthKey, error) {
    rows, err := s.db.Query("SELECT key, created_at, expires_at, is_active FROM auth_keys ORDER BY created_at DESC")
    if err != nil {
        return nil, fmt.Errorf("failed to list keys: %v", err)
    }
    defer rows.Close()
    
    var keys []*models.AuthKey
    for rows.Next() {
        var key models.AuthKey
        if err := rows.Scan(&key.Key, &key.CreatedAt, &key.ExpiresAt, &key.IsActive); err != nil {
            return nil, fmt.Errorf("failed to scan key: %v", err)
        }
        keys = append(keys, &key)
    }
    
    return keys, nil
}

// PostgreSQLIPSetStorage
type PostgreSQLIPSetStorage struct {
    db     *sql.DB
    nextID int
}

func NewPostgreSQLIPSetStorage(cfg *config.Config) (*PostgreSQLIPSetStorage, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.PostgreSQLHost,
        cfg.PostgreSQLPort,
        cfg.PostgreSQLUsername,
        cfg.PostgreSQLPassword,
        cfg.PostgreSQLDatabase,
    )
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to postgresql: %v", err)
    }
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping postgresql: %v", err)
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS ipset_records (
            id INTEGER PRIMARY KEY CHECK (id >= 100000 AND id <= 999999),
            set_name VARCHAR(255) NOT NULL,
            ip VARCHAR(45) NOT NULL,
            cidr VARCHAR(45),
            port INTEGER,
            protocol VARCHAR(10),
            description TEXT,
            context TEXT NOT NULL,
            set_type VARCHAR(50),
            set_options TEXT,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create ipset_records table: %v", err)
    }
    
    // Создаем индексы для поиска
    _, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_ipset_records_set_name ON ipset_records(set_name);
        CREATE INDEX IF NOT EXISTS idx_ipset_records_ip ON ipset_records(ip);
        CREATE INDEX IF NOT EXISTS idx_ipset_records_context ON ipset_records USING gin(to_tsvector('english', context));
        CREATE INDEX IF NOT EXISTS idx_ipset_records_description ON ipset_records USING gin(to_tsvector('english', description));
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create indexes: %v", err)
    }
    
    // Создаем функцию для автоматического обновления updated_at
    _, err = db.Exec(`
        CREATE OR REPLACE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $$
        BEGIN
            NEW.updated_at = CURRENT_TIMESTAMP;
            RETURN NEW;
        END;
        $$ language 'plpgsql';
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create function: %v", err)
    }
    
    // Создаем триггер для автоматического обновления updated_at
    _, err = db.Exec(`
        DROP TRIGGER IF EXISTS update_ipset_records_updated_at ON ipset_records;
        CREATE TRIGGER update_ipset_records_updated_at
            BEFORE UPDATE ON ipset_records
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create trigger: %v", err)
    }
    
    // Получаем максимальный ID для определения следующего доступного
    var maxID sql.NullInt64
    err = db.QueryRow("SELECT MAX(id) FROM ipset_records").Scan(&maxID)
    if err != nil {
        return nil, fmt.Errorf("failed to get max ID: %v", err)
    }
    
    nextID := 100000
    if maxID.Valid {
        nextID = int(maxID.Int64) + 1
        if nextID < 100000 {
            nextID = 100000
        }
    }
    
    return &PostgreSQLIPSetStorage{
        db:     db,
        nextID: nextID,
    }, nil
}

func (s *PostgreSQLIPSetStorage) getNextID() (int, error) {
    // Ищем первый свободный ID в диапазоне 100000-999999
    var id int
    err := s.db.QueryRow(`
        SELECT generate_series
        FROM generate_series(100000, 999999) AS generate_series
        WHERE generate_series NOT IN (SELECT id FROM ipset_records)
        ORDER BY generate_series
        LIMIT 1
    `).Scan(&id)
    
    if err == sql.ErrNoRows {
        return 0, fmt.Errorf("no available IDs in range 100000-999999")
    }
    if err != nil {
        return 0, fmt.Errorf("failed to get next ID: %v", err)
    }
    
    return id, nil
}

func (s *PostgreSQLIPSetStorage) Create(record *models.IPSetRecord) error {
    // Получаем следующий доступный ID
    id, err := s.getNextID()
    if err != nil {
        return err
    }
    
    record.ID = id
    now := time.Now()
    record.CreatedAt = now
    record.UpdatedAt = now
    
    _, err = s.db.Exec(`
        INSERT INTO ipset_records 
        (id, set_name, ip, cidr, port, protocol, description, context, set_type, set_options, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `,
        record.ID, record.SetName, record.IP, record.CIDR, record.Port, record.Protocol,
        record.Description, record.Context, record.SetType, record.SetOptions,
        record.CreatedAt, record.UpdatedAt,
    )
    
    if err != nil {
        return fmt.Errorf("failed to create record: %v", err)
    }
    
    return nil
}

func (s *PostgreSQLIPSetStorage) GetByID(id int) (*models.IPSetRecord, error) {
    var record models.IPSetRecord
    err := s.db.QueryRow(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE id = $1
    `, id).Scan(
        &record.ID, &record.SetName, &record.IP, &record.CIDR, &record.Port, &record.Protocol,
        &record.Description, &record.Context, &record.SetType, &record.SetOptions,
        &record.CreatedAt, &record.UpdatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("record with id %d not found", id)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get record: %v", err)
    }
    
    return &record, nil
}

func (s *PostgreSQLIPSetStorage) GetAll() ([]*models.IPSetRecord, error) {
    rows, err := s.db.Query(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
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

func (s *PostgreSQLIPSetStorage) GetBySetName(setName string) ([]*models.IPSetRecord, error) {
    rows, err := s.db.Query(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE set_name = $1
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

func (s *PostgreSQLIPSetStorage) GetAllSets() ([]*models.IPSetSet, error) {
    rows, err := s.db.Query(`
        SELECT 
            set_name, 
            set_type, 
            set_options, 
            MIN(created_at) as created_at,
            MAX(updated_at) as updated_at,
            COUNT(*) as record_count
        FROM ipset_records
        GROUP BY set_name, set_type, set_options
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
        var recordCount int
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

func (s *PostgreSQLIPSetStorage) Update(id int, record *models.IPSetRecord) error {
    result, err := s.db.Exec(`
        UPDATE ipset_records
        SET set_name = $1, ip = $2, cidr = $3, port = $4, protocol = $5, 
            description = $6, context = $7, set_type = $8, set_options = $9
        WHERE id = $10
    `,
        record.SetName, record.IP, record.CIDR, record.Port, record.Protocol,
        record.Description, record.Context, record.SetType, record.SetOptions, id,
    )
    
    if err != nil {
        return fmt.Errorf("failed to update record: %v", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %v", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("record with id %d not found", id)
    }
    
    return nil
}

func (s *PostgreSQLIPSetStorage) Delete(id int) error {
    result, err := s.db.Exec("DELETE FROM ipset_records WHERE id = $1", id)
    if err != nil {
        return fmt.Errorf("failed to delete record: %v", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %v", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("record with id %d not found", id)
    }
    
    return nil
}

func (s *PostgreSQLIPSetStorage) DeleteSet(setName string) error {
    result, err := s.db.Exec("DELETE FROM ipset_records WHERE set_name = $1", setName)
    if err != nil {
        return fmt.Errorf("failed to delete set: %v", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %v", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("set %s not found", setName)
    }
    
    return nil
}

func (s *PostgreSQLIPSetStorage) Search(query string) ([]*models.IPSetRecord, error) {
    // Используем полнотекстовый поиск PostgreSQL для лучших результатов
    rows, err := s.db.Query(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE 
            to_tsvector('english', COALESCE(context, '')) @@ plainto_tsquery('english', $1)
            OR to_tsvector('english', COALESCE(description, '')) @@ plainto_tsquery('english', $1)
            OR context ILIKE '%' || $1 || '%'
            OR description ILIKE '%' || $1 || '%'
            OR ip ILIKE '%' || $1 || '%'
            OR set_name ILIKE '%' || $1 || '%'
        ORDER BY 
            CASE 
                WHEN set_name = $1 THEN 1
                WHEN ip = $1 THEN 2
                WHEN context ILIKE $1 THEN 3
                WHEN description ILIKE $1 THEN 4
                ELSE 5
            END,
            id
    `, query)
    
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