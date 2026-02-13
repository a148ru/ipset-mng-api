// internal/storage/mysql_storage.go
package storage

import (
    "database/sql"
    "fmt"
    "ipset-api/internal/config"
    "ipset-api/internal/models"
    
    _ "github.com/go-sql-driver/mysql"
)

type MySQLKeyStorage struct {
    db *sql.DB
}

func NewMySQLKeyStorage(cfg *config.Config) (*MySQLKeyStorage, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
        cfg.MySQLUsername,
        cfg.MySQLPassword,
        cfg.MySQLHost,
        cfg.MySQLPort,
        cfg.MySQLDatabase,
    )
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS auth_keys (
            key VARCHAR(255) PRIMARY KEY,
            created_at DATETIME,
            expires_at DATETIME,
            is_active BOOLEAN
        )
    `)
    if err != nil {
        return nil, err
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
        return nil, err
    }
    
    return &authKey, nil
}

func (s *MySQLKeyStorage) SaveKey(key *models.AuthKey) error {
    _, err := s.db.Exec(
        "INSERT INTO auth_keys (key, created_at, expires_at, is_active) VALUES (?, ?, ?, ?)",
        key.Key, key.CreatedAt, key.ExpiresAt, key.IsActive,
    )
    return err
}

func (s *MySQLKeyStorage) DeleteKey(key string) error {
    _, err := s.db.Exec("DELETE FROM auth_keys WHERE key = ?", key)
    return err
}

func (s *MySQLKeyStorage) ListKeys() ([]*models.AuthKey, error) {
    rows, err := s.db.Query("SELECT key, created_at, expires_at, is_active FROM auth_keys")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var keys []*models.AuthKey
    for rows.Next() {
        var key models.AuthKey
        if err := rows.Scan(&key.Key, &key.CreatedAt, &key.ExpiresAt, &key.IsActive); err != nil {
            return nil, err
        }
        keys = append(keys, &key)
    }
    
    return keys, nil
}

type MySQLIPSetStorage struct {
    db *sql.DB
}

func NewMySQLIPSetStorage(cfg *config.Config) (*MySQLIPSetStorage, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
        cfg.MySQLUsername,
        cfg.MySQLPassword,
        cfg.MySQLHost,
        cfg.MySQLPort,
        cfg.MySQLDatabase,
    )
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS ipset_records (
            id INT PRIMARY KEY,
            ip VARCHAR(45),
            cidr VARCHAR(45),
            port INT,
            protocol VARCHAR(10),
            description TEXT,
            context TEXT,
            created_at DATETIME,
            updated_at DATETIME
        )
    `)
    if err != nil {
        return nil, err
    }
    
    return &MySQLIPSetStorage{db: db}, nil
}

// Implement MySQLIPSetStorage methods similarly to file storage but with SQL queries