package storage

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    
    _ "github.com/lib/pq"
)

type PostgreSQLKeyStorage struct {
    db *sql.DB
}

func NewPostgreSQLKeyStorage(cfg *config.Config) (*PostgreSQLKeyStorage, error) {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.PostgreSQLHost,
        cfg.PostgreSQLPort,
        cfg.PostgreSQLUsername,
        cfg.PostgreSQLPassword,
        cfg.PostgreSQLDatabase,
    )
    
    db, err := sql.Open("postgres", connStr)
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
            created_at TIMESTAMP,
            expires_at TIMESTAMP,
            is_active BOOLEAN
        )
    `)
    if err != nil {
        return nil, err
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
        return nil, err
    }
    
    return &authKey, nil
}

func (s *PostgreSQLKeyStorage) SaveKey(key *models.AuthKey) error {
    _, err := s.db.Exec(
        "INSERT INTO auth_keys (key, created_at, expires_at, is_active) VALUES ($1, $2, $3, $4)",
        key.Key, key.CreatedAt, key.ExpiresAt, key.IsActive,
    )
    return err
}

func (s *PostgreSQLKeyStorage) DeleteKey(key string) error {
    _, err := s.db.Exec("DELETE FROM auth_keys WHERE key = $1", key)
    return err
}

func (s *PostgreSQLKeyStorage) ListKeys() ([]*models.AuthKey, error) {
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

type PostgreSQLIPSetStorage struct {
    db *sql.DB
}

func NewPostgreSQLIPSetStorage(cfg *config.Config) (*PostgreSQLIPSetStorage, error) {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.PostgreSQLHost,
        cfg.PostgreSQLPort,
        cfg.PostgreSQLUsername,
        cfg.PostgreSQLPassword,
        cfg.PostgreSQLDatabase,
    )
    
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    // Создаем таблицы если не существуют
    if err := createPostgreSQLTables(db); err != nil {
        return nil, err
    }
    
    return &PostgreSQLIPSetStorage{db: db}, nil
}

func createPostgreSQLTables(db *sql.DB) error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS ipset_records (
            id INTEGER PRIMARY KEY,
            ip VARCHAR(45),
            cidr VARCHAR(45),
            port INTEGER,
            protocol VARCHAR(10),
            description TEXT,
            context TEXT,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        )`,
        `CREATE TABLE IF NOT EXISTS ipsets (
            name VARCHAR(255) PRIMARY KEY,
            type VARCHAR(50),
            family VARCHAR(10),
            hashsize INTEGER,
            maxelem INTEGER,
            description TEXT,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        )`,
        `CREATE TABLE IF NOT EXISTS ipset_entries (
            id SERIAL PRIMARY KEY,
            ipset_name VARCHAR(255),
            value TEXT,
            comment TEXT,
            created_at TIMESTAMP,
            FOREIGN KEY (ipset_name) REFERENCES ipsets(name) ON DELETE CASCADE
        )`,
        `CREATE TABLE IF NOT EXISTS iptables_rules (
            id INTEGER PRIMARY KEY,
            chain VARCHAR(255),
            interface VARCHAR(255),
            protocol VARCHAR(50),
            src_sets TEXT,
            dst_sets TEXT,
            action VARCHAR(50),
            description TEXT,
            position INTEGER,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        )`,
    }
    
    for _, query := range queries {
        if _, err := db.Exec(query); err != nil {
            return err
        }
    }
    
    return nil
}

// IPSetRecord methods
func (s *PostgreSQLIPSetStorage) CreateRecord(record *models.IPSetRecord) error {
    record.CreatedAt = time.Now()
    record.UpdatedAt = time.Now()
    
    _, err := s.db.Exec(
        "INSERT INTO ipset_records (id, ip, cidr, port, protocol, description, context, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
        record.ID, record.IP, record.CIDR, record.Port, record.Protocol, record.Description, record.Context, record.CreatedAt, record.UpdatedAt,
    )
    return err
}

func (s *PostgreSQLIPSetStorage) GetRecordByID(id int) (*models.IPSetRecord, error) {
    var record models.IPSetRecord
    err := s.db.QueryRow(
        "SELECT id, ip, cidr, port, protocol, description, context, created_at, updated_at FROM ipset_records WHERE id = $1",
        id,
    ).Scan(&record.ID, &record.IP, &record.CIDR, &record.Port, &record.Protocol, &record.Description, &record.Context, &record.CreatedAt, &record.UpdatedAt)
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("record with id %d not found", id)
    }
    if err != nil {
        return nil, err
    }
    
    return &record, nil
}

func (s *PostgreSQLIPSetStorage) GetAllRecords() ([]*models.IPSetRecord, error) {
    rows, err := s.db.Query("SELECT id, ip, cidr, port, protocol, description, context, created_at, updated_at FROM ipset_records")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var records []*models.IPSetRecord
    for rows.Next() {
        var record models.IPSetRecord
        if err := rows.Scan(&record.ID, &record.IP, &record.CIDR, &record.Port, &record.Protocol, &record.Description, &record.Context, &record.CreatedAt, &record.UpdatedAt); err != nil {
            return nil, err
        }
        records = append(records, &record)
    }
    
    return records, nil
}

func (s *PostgreSQLIPSetStorage) UpdateRecord(id int, record *models.IPSetRecord) error {
    record.UpdatedAt = time.Now()
    
    _, err := s.db.Exec(
        "UPDATE ipset_records SET ip = $1, cidr = $2, port = $3, protocol = $4, description = $5, context = $6, updated_at = $7 WHERE id = $8",
        record.IP, record.CIDR, record.Port, record.Protocol, record.Description, record.Context, record.UpdatedAt, id,
    )
    return err
}

func (s *PostgreSQLIPSetStorage) DeleteRecord(id int) error {
    _, err := s.db.Exec("DELETE FROM ipset_records WHERE id = $1", id)
    return err
}

func (s *PostgreSQLIPSetStorage) SearchRecords(context string) ([]*models.IPSetRecord, error) {
    rows, err := s.db.Query(
        "SELECT id, ip, cidr, port, protocol, description, context, created_at, updated_at FROM ipset_records WHERE context ILIKE $1 OR description ILIKE $1",
        "%"+context+"%",
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var records []*models.IPSetRecord
    for rows.Next() {
        var record models.IPSetRecord
        if err := rows.Scan(&record.ID, &record.IP, &record.CIDR, &record.Port, &record.Protocol, &record.Description, &record.Context, &record.CreatedAt, &record.UpdatedAt); err != nil {
            return nil, err
        }
        records = append(records, &record)
    }
    
    return records, nil
}

// IPSet methods
func (s *PostgreSQLIPSetStorage) CreateIPSet(set *models.IPSet) error {
    set.CreatedAt = time.Now()
    set.UpdatedAt = time.Now()
    
    _, err := s.db.Exec(
        "INSERT INTO ipsets (name, type, family, hashsize, maxelem, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
        set.Name, set.Type, set.Family, set.HashSize, set.MaxElem, set.Description, set.CreatedAt, set.UpdatedAt,
    )
    return err
}

func (s *PostgreSQLIPSetStorage) GetIPSet(name string) (*models.IPSet, error) {
    var set models.IPSet
    err := s.db.QueryRow(
        "SELECT name, type, family, hashsize, maxelem, description, created_at, updated_at FROM ipsets WHERE name = $1",
        name,
    ).Scan(&set.Name, &set.Type, &set.Family, &set.HashSize, &set.MaxElem, &set.Description, &set.CreatedAt, &set.UpdatedAt)
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("ipset %s not found", name)
    }
    if err != nil {
        return nil, err
    }
    
    // Загружаем entries
    entries, err := s.GetIPSetEntries(name)
    if err != nil {
        return nil, err
    }
    set.Entries = make([]models.IPSetEntry, len(entries))
    for i, entry := range entries {
        set.Entries[i] = *entry
    }
    
    return &set, nil
}

func (s *PostgreSQLIPSetStorage) GetAllIPSets() ([]*models.IPSet, error) {
    rows, err := s.db.Query("SELECT name, type, family, hashsize, maxelem, description, created_at, updated_at FROM ipsets")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var sets []*models.IPSet
    for rows.Next() {
        var set models.IPSet
        if err := rows.Scan(&set.Name, &set.Type, &set.Family, &set.HashSize, &set.MaxElem, &set.Description, &set.CreatedAt, &set.UpdatedAt); err != nil {
            return nil, err
        }
        sets = append(sets, &set)
    }
    
    return sets, nil
}

func (s *PostgreSQLIPSetStorage) UpdateIPSet(name string, set *models.IPSet) error {
    set.UpdatedAt = time.Now()
    
    _, err := s.db.Exec(
        "UPDATE ipsets SET name = $1, type = $2, family = $3, hashsize = $4, maxelem = $5, description = $6, updated_at = $7 WHERE name = $8",
        set.Name, set.Type, set.Family, set.HashSize, set.MaxElem, set.Description, set.UpdatedAt, name,
    )
    return err
}

func (s *PostgreSQLIPSetStorage) DeleteIPSet(name string) error {
    _, err := s.db.Exec("DELETE FROM ipsets WHERE name = $1", name)
    return err
}

func (s *PostgreSQLIPSetStorage) AddIPSetEntry(setName string, entry *models.IPSetEntry) error {
    entry.CreatedAt = time.Now()
    
    err := s.db.QueryRow(
        "INSERT INTO ipset_entries (ipset_name, value, comment, created_at) VALUES ($1, $2, $3, $4) RETURNING id",
        setName, entry.Value, entry.Comment, entry.CreatedAt,
    ).Scan(&entry.ID)
    
    return err
}

func (s *PostgreSQLIPSetStorage) RemoveIPSetEntry(entryID int) error {
    _, err := s.db.Exec("DELETE FROM ipset_entries WHERE id = $1", entryID)
    return err
}

func (s *PostgreSQLIPSetStorage) GetIPSetEntries(setName string) ([]*models.IPSetEntry, error) {
    rows, err := s.db.Query("SELECT id, ipset_name, value, comment, created_at FROM ipset_entries WHERE ipset_name = $1", setName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var entries []*models.IPSetEntry
    for rows.Next() {
        var entry models.IPSetEntry
        if err := rows.Scan(&entry.ID, &entry.IPSetName, &entry.Value, &entry.Comment, &entry.CreatedAt); err != nil {
            return nil, err
        }
        entries = append(entries, &entry)
    }
    
    return entries, nil
}

func (s *PostgreSQLIPSetStorage) SearchIPSets(query string) ([]*models.IPSet, error) {
    rows, err := s.db.Query(
        "SELECT name, type, family, hashsize, maxelem, description, created_at, updated_at FROM ipsets WHERE name ILIKE $1 OR description ILIKE $1",
        "%"+query+"%",
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var sets []*models.IPSet
    for rows.Next() {
        var set models.IPSet
        if err := rows.Scan(&set.Name, &set.Type, &set.Family, &set.HashSize, &set.MaxElem, &set.Description, &set.CreatedAt, &set.UpdatedAt); err != nil {
            return nil, err
        }
        sets = append(sets, &set)
    }
    
    return sets, nil
}

// IPTables methods
func (s *PostgreSQLIPSetStorage) CreateIPTablesRule(rule *models.IPTablesRule) error {
    rule.CreatedAt = time.Now()
    rule.UpdatedAt = time.Now()
    
    // Сериализуем массивы в JSON
    srcSetsJSON, _ := json.Marshal(rule.SrcSets)
    dstSetsJSON, _ := json.Marshal(rule.DstSets)
    
    _, err := s.db.Exec(
        "INSERT INTO iptables_rules (id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
        rule.ID, rule.Chain, rule.Interface, rule.Protocol, string(srcSetsJSON), string(dstSetsJSON), rule.Action, rule.Description, rule.Position, rule.CreatedAt, rule.UpdatedAt,
    )
    return err
}

func (s *PostgreSQLIPSetStorage) GetIPTablesRule(id int) (*models.IPTablesRule, error) {
    var rule models.IPTablesRule
    var srcSetsJSON, dstSetsJSON string
    
    err := s.db.QueryRow(
        "SELECT id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at FROM iptables_rules WHERE id = $1",
        id,
    ).Scan(&rule.ID, &rule.Chain, &rule.Interface, &rule.Protocol, &srcSetsJSON, &dstSetsJSON, &rule.Action, &rule.Description, &rule.Position, &rule.CreatedAt, &rule.UpdatedAt)
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("rule with id %d not found", id)
    }
    if err != nil {
        return nil, err
    }
    
    // Десериализуем JSON обратно в массивы
    json.Unmarshal([]byte(srcSetsJSON), &rule.SrcSets)
    json.Unmarshal([]byte(dstSetsJSON), &rule.DstSets)
    
    return &rule, nil
}

func (s *PostgreSQLIPSetStorage) GetAllIPTablesRules() ([]*models.IPTablesRule, error) {
    rows, err := s.db.Query("SELECT id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at FROM iptables_rules")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var rules []*models.IPTablesRule
    for rows.Next() {
        var rule models.IPTablesRule
        var srcSetsJSON, dstSetsJSON string
        
        if err := rows.Scan(&rule.ID, &rule.Chain, &rule.Interface, &rule.Protocol, &srcSetsJSON, &dstSetsJSON, &rule.Action, &rule.Description, &rule.Position, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
            return nil, err
        }
        
        json.Unmarshal([]byte(srcSetsJSON), &rule.SrcSets)
        json.Unmarshal([]byte(dstSetsJSON), &rule.DstSets)
        
        rules = append(rules, &rule)
    }
    
    return rules, nil
}

func (s *PostgreSQLIPSetStorage) UpdateIPTablesRule(id int, rule *models.IPTablesRule) error {
    rule.UpdatedAt = time.Now()
    
    srcSetsJSON, _ := json.Marshal(rule.SrcSets)
    dstSetsJSON, _ := json.Marshal(rule.DstSets)
    
    _, err := s.db.Exec(
        "UPDATE iptables_rules SET chain = $1, interface = $2, protocol = $3, src_sets = $4, dst_sets = $5, action = $6, description = $7, position = $8, updated_at = $9 WHERE id = $10",
        rule.Chain, rule.Interface, rule.Protocol, string(srcSetsJSON), string(dstSetsJSON), rule.Action, rule.Description, rule.Position, rule.UpdatedAt, id,
    )
    return err
}

func (s *PostgreSQLIPSetStorage) DeleteIPTablesRule(id int) error {
    _, err := s.db.Exec("DELETE FROM iptables_rules WHERE id = $1", id)
    return err
}

func (s *PostgreSQLIPSetStorage) ReorderIPTablesRule(id int, newPosition int) error {
    _, err := s.db.Exec("UPDATE iptables_rules SET position = $1, updated_at = $2 WHERE id = $3", newPosition, time.Now(), id)
    return err
}

func (s *PostgreSQLIPSetStorage) SearchIPTablesRules(query string) ([]*models.IPTablesRule, error) {
    rows, err := s.db.Query(
        "SELECT id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at FROM iptables_rules WHERE chain ILIKE $1 OR description ILIKE $1",
        "%"+query+"%",
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var rules []*models.IPTablesRule
    for rows.Next() {
        var rule models.IPTablesRule
        var srcSetsJSON, dstSetsJSON string
        
        if err := rows.Scan(&rule.ID, &rule.Chain, &rule.Interface, &rule.Protocol, &srcSetsJSON, &dstSetsJSON, &rule.Action, &rule.Description, &rule.Position, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
            return nil, err
        }
        
        json.Unmarshal([]byte(srcSetsJSON), &rule.SrcSets)
        json.Unmarshal([]byte(dstSetsJSON), &rule.DstSets)
        
        rules = append(rules, &rule)
    }
    
    return rules, nil
}