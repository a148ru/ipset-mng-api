package storage

import (
    "database/sql"
    "fmt"
     "time"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    
    _ "github.com/go-sql-driver/mysql"
)

// MySQLKeyStorage - реализация для хранения ключей в MySQL
type MySQLKeyStorage struct {
    db *sql.DB
}
func NewMySQLKeyStorage(cfg *config.Config) (*MySQLKeyStorage, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
        cfg.MySQLUsername,
        cfg.MySQLPassword,
        cfg.MySQLHost,
        cfg.MySQLPort,
        cfg.MySQLDatabase,
    )
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to mysql: %v", err)
    }
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping mysql: %v", err)
    }
    
    // Создаем базу данных если не существует
    _, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.MySQLDatabase))
    if err != nil {
        return nil, fmt.Errorf("failed to create database: %v", err)
    }
    
    // Используем созданную базу данных
    _, err = db.Exec(fmt.Sprintf("USE %s", cfg.MySQLDatabase))
    if err != nil {
        return nil, fmt.Errorf("failed to use database: %v", err)
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS auth_keys (
            key VARCHAR(255) PRIMARY KEY,
            created_at DATETIME,
            expires_at DATETIME,
            is_active BOOLEAN
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create auth_keys table: %v", err)
    }
    
    return &MySQLKeyStorage{db: db}, nil
}
func (s *MySQLKeyStorage) GetKey(key string) (*models.AuthKey, error) {
    var authKey models.AuthKey
    err := s.db.QueryRow(
        "SELECT key, created_at, expires_at, is_active FROM auth_keys WHERE key = ?",
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

func (s *MySQLKeyStorage) SaveKey(key *models.AuthKey) error {
    _, err := s.db.Exec(
        `INSERT INTO auth_keys (key, created_at, expires_at, is_active) 
         VALUES (?, ?, ?, ?)
         ON DUPLICATE KEY UPDATE 
         created_at = VALUES(created_at),
         expires_at = VALUES(expires_at),
         is_active = VALUES(is_active)`,
        key.Key, key.CreatedAt, key.ExpiresAt, key.IsActive,
    )
    if err != nil {
        return fmt.Errorf("failed to save key: %v", err)
    }
    return nil
}

func (s *MySQLKeyStorage) DeleteKey(key string) error {
    _, err := s.db.Exec("DELETE FROM auth_keys WHERE key = ?", key)
    if err != nil {
        return fmt.Errorf("failed to delete key: %v", err)
    }
    return nil
}

func (s *MySQLKeyStorage) ListKeys() ([]*models.AuthKey, error) {
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

// MySQLIPSetStorage - реализация для хранения ipset записей в MySQL
type MySQLIPSetStorage struct {
    db *sql.DB
}

func NewMySQLIPSetStorage(cfg *config.Config) (*MySQLIPSetStorage, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
        cfg.MySQLUsername,
        cfg.MySQLPassword,
        cfg.MySQLHost,
        cfg.MySQLPort,
        cfg.MySQLDatabase,
    )
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to mysql: %v", err)
    }
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping mysql: %v", err)
    }
    
    // Создаем базу данных если не существует
    _, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.MySQLDatabase))
    if err != nil {
        return nil, fmt.Errorf("failed to create database: %v", err)
    }
    
    // Используем созданную базу данных
    _, err = db.Exec(fmt.Sprintf("USE %s", cfg.MySQLDatabase))
    if err != nil {
        return nil, fmt.Errorf("failed to use database: %v", err)
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS ipset_records (
            id INT PRIMARY KEY,
            set_name VARCHAR(255) NOT NULL,
            ip VARCHAR(45) NOT NULL,
            cidr VARCHAR(45),
            port INT,
            protocol VARCHAR(10),
            description TEXT,
            context TEXT NOT NULL,
            set_type VARCHAR(50),
            set_options TEXT,
            created_at DATETIME,
            updated_at DATETIME,
            INDEX idx_set_name (set_name),
            INDEX idx_ip (ip),
            INDEX idx_context (context(255)),
            CONSTRAINT chk_id CHECK (id >= 100000 AND id <= 999999)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create ipset_records table: %v", err)
    }
    
    // Создаем триггер для автоматического обновления updated_at
    _, err = db.Exec(`
        DROP TRIGGER IF EXISTS update_ipset_records_updated_at
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to drop trigger: %v", err)
    }
    
    _, err = db.Exec(`
        CREATE TRIGGER update_ipset_records_updated_at
        BEFORE UPDATE ON ipset_records
        FOR EACH ROW
        SET NEW.updated_at = NOW()
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create trigger: %v", err)
    }
    
    return &MySQLIPSetStorage{db: db}, nil
}

func (s *MySQLIPSetStorage) getNextID() (int, error) {
    var maxID sql.NullInt64
    err := s.db.QueryRow("SELECT MAX(id) FROM ipset_records").Scan(&maxID)
    if err != nil {
        return 0, fmt.Errorf("failed to get max ID: %v", err)
    }
    
    nextID := 100000
    if maxID.Valid {
        nextID = int(maxID.Int64) + 1
        if nextID < 100000 {
            nextID = 100000
        }
    }
    
    if nextID > 999999 {
        rows, err := s.db.Query(`
            SELECT t1.id + 1 AS next_id
            FROM ipset_records t1
            LEFT JOIN ipset_records t2 ON t1.id + 1 = t2.id
            WHERE t2.id IS NULL AND t1.id >= 100000 AND t1.id < 999999
            ORDER BY t1.id
            LIMIT 1
        `)
        if err != nil {
            return 0, fmt.Errorf("failed to find free ID: %v", err)
        }
        defer rows.Close()
        
        if rows.Next() {
            err = rows.Scan(&nextID)
            if err != nil {
                return 0, fmt.Errorf("failed to scan free ID: %v", err)
            }
        } else {
            return 0, fmt.Errorf("no available IDs in range 100000-999999")
        }
    }
    
    return nextID, nil
}

func (s *MySQLIPSetStorage) Create(record *models.IPSetRecord) error {
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
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

func (s *MySQLIPSetStorage) GetByID(id int) (*models.IPSetRecord, error) {
    var record models.IPSetRecord
    err := s.db.QueryRow(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE id = ?
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

func (s *MySQLIPSetStorage) GetAll() ([]*models.IPSetRecord, error) {
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

func (s *MySQLIPSetStorage) GetBySetName(setName string) ([]*models.IPSetRecord, error) {
    rows, err := s.db.Query(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE set_name = ?
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

func (s *MySQLIPSetStorage) GetAllSets() ([]*models.IPSetSet, error) {
    rows, err := s.db.Query(`
        SELECT set_name, set_type, set_options, 
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

func (s *MySQLIPSetStorage) Update(id int, record *models.IPSetRecord) error {
    result, err := s.db.Exec(`
        UPDATE ipset_records
        SET set_name = ?, ip = ?, cidr = ?, port = ?, protocol = ?, 
            description = ?, context = ?, set_type = ?, set_options = ?,
            updated_at = NOW()
        WHERE id = ?
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

func (s *MySQLIPSetStorage) Delete(id int) error {
    result, err := s.db.Exec("DELETE FROM ipset_records WHERE id = ?", id)
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

func (s *MySQLIPSetStorage) DeleteSet(setName string) error {
    result, err := s.db.Exec("DELETE FROM ipset_records WHERE set_name = ?", setName)
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

func (s *MySQLIPSetStorage) Search(query string) ([]*models.IPSetRecord, error) {
    searchPattern := "%" + query + "%"
    rows, err := s.db.Query(`
        SELECT id, set_name, ip, cidr, port, protocol, description, context, 
               set_type, set_options, created_at, updated_at
        FROM ipset_records
        WHERE 
            context LIKE ? OR
            description LIKE ? OR
            ip LIKE ? OR
            set_name LIKE ?
        ORDER BY 
            CASE 
                WHEN set_name = ? THEN 1
                WHEN ip = ? THEN 2
                WHEN context LIKE ? THEN 3
                ELSE 4
            END,
            id
    `, searchPattern, searchPattern, searchPattern, searchPattern,
        query, query, searchPattern)
    
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