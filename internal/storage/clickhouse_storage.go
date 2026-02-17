package storage

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    
    _ "github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseKeyStorage struct {
    db *sql.DB
}

func NewClickHouseKeyStorage(cfg *config.Config) (*ClickHouseKeyStorage, error) {
    dsn := fmt.Sprintf("tcp://%s:%s?username=%s&password=%s&database=%s",
        cfg.ClickHouseHost,
        cfg.ClickHousePort,
        cfg.ClickHouseUsername,
        cfg.ClickHousePassword,
        cfg.ClickHouseDatabase,
    )
    
    db, err := sql.Open("clickhouse", dsn)
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    // Создаем таблицу если не существует
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS auth_keys (
            key String,
            created_at DateTime,
            expires_at DateTime,
            is_active UInt8
        ) ENGINE = MergeTree() ORDER BY key
    `)
    if err != nil {
        return nil, err
    }
    
    return &ClickHouseKeyStorage{db: db}, nil
}

func (s *ClickHouseKeyStorage) GetKey(key string) (*models.AuthKey, error) {
    var authKey models.AuthKey
    var isActive uint8
    
    err := s.db.QueryRow(
        "SELECT key, created_at, expires_at, is_active FROM auth_keys WHERE key = ?",
        key,
    ).Scan(&authKey.Key, &authKey.CreatedAt, &authKey.ExpiresAt, &isActive)
    
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    
    authKey.IsActive = isActive == 1
    return &authKey, nil
}

func (s *ClickHouseKeyStorage) SaveKey(key *models.AuthKey) error {
    isActive := uint8(0)
    if key.IsActive {
        isActive = 1
    }
    
    _, err := s.db.Exec(
        "INSERT INTO auth_keys (key, created_at, expires_at, is_active) VALUES (?, ?, ?, ?)",
        key.Key, key.CreatedAt, key.ExpiresAt, isActive,
    )
    return err
}

func (s *ClickHouseKeyStorage) DeleteKey(key string) error {
    _, err := s.db.Exec("ALTER TABLE auth_keys DELETE WHERE key = ?", key)
    return err
}

func (s *ClickHouseKeyStorage) ListKeys() ([]*models.AuthKey, error) {
    rows, err := s.db.Query("SELECT key, created_at, expires_at, is_active FROM auth_keys")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var keys []*models.AuthKey
    for rows.Next() {
        var key models.AuthKey
        var isActive uint8
        if err := rows.Scan(&key.Key, &key.CreatedAt, &key.ExpiresAt, &isActive); err != nil {
            return nil, err
        }
        key.IsActive = isActive == 1
        keys = append(keys, &key)
    }
    
    return keys, nil
}

type ClickHouseIPSetStorage struct {
    db *sql.DB
}

func NewClickHouseIPSetStorage(cfg *config.Config) (*ClickHouseIPSetStorage, error) {
    dsn := fmt.Sprintf("tcp://%s:%s?username=%s&password=%s&database=%s",
        cfg.ClickHouseHost,
        cfg.ClickHousePort,
        cfg.ClickHouseUsername,
        cfg.ClickHousePassword,
        cfg.ClickHouseDatabase,
    )
    
    db, err := sql.Open("clickhouse", dsn)
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    // Создаем таблицы если не существуют
    if err := createClickHouseTables(db); err != nil {
        return nil, err
    }
    
    return &ClickHouseIPSetStorage{db: db}, nil
}

func createClickHouseTables(db *sql.DB) error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS ipset_records (
            id UInt32,
            ip String,
            cidr String,
            port UInt16,
            protocol String,
            description String,
            context String,
            created_at DateTime,
            updated_at DateTime
        ) ENGINE = MergeTree() ORDER BY id`,
        
        `CREATE TABLE IF NOT EXISTS ipsets (
            name String,
            type String,
            family String,
            hashsize UInt32,
            maxelem UInt32,
            description String,
            created_at DateTime,
            updated_at DateTime
        ) ENGINE = MergeTree() ORDER BY name`,
        
        `CREATE TABLE IF NOT EXISTS ipset_entries (
            id UInt32,
            ipset_name String,
            value String,
            comment String,
            created_at DateTime
        ) ENGINE = MergeTree() ORDER BY id`,
        
        `CREATE TABLE IF NOT EXISTS iptables_rules (
            id UInt32,
            chain String,
            interface String,
            protocol String,
            src_sets String,
            dst_sets String,
            action String,
            description String,
            position UInt32,
            created_at DateTime,
            updated_at DateTime
        ) ENGINE = MergeTree() ORDER BY id`,
    }
    
    for _, query := range queries {
        if _, err := db.Exec(query); err != nil {
            return err
        }
    }
    
    return nil
}

// IPSetRecord methods
func (s *ClickHouseIPSetStorage) CreateRecord(record *models.IPSetRecord) error {
    record.CreatedAt = time.Now()
    record.UpdatedAt = time.Now()
    
    _, err := s.db.Exec(
        "INSERT INTO ipset_records (id, ip, cidr, port, protocol, description, context, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
        record.ID, record.IP, record.CIDR, record.Port, record.Protocol, record.Description, record.Context, record.CreatedAt, record.UpdatedAt,
    )
    return err
}

func (s *ClickHouseIPSetStorage) GetRecordByID(id int) (*models.IPSetRecord, error) {
    var record models.IPSetRecord
    err := s.db.QueryRow(
        "SELECT id, ip, cidr, port, protocol, description, context, created_at, updated_at FROM ipset_records WHERE id = ?",
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

func (s *ClickHouseIPSetStorage) GetAllRecords() ([]*models.IPSetRecord, error) {
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

func (s *ClickHouseIPSetStorage) UpdateRecord(id int, record *models.IPSetRecord) error {
    record.UpdatedAt = time.Now()
    
    // ClickHouse не поддерживает UPDATE напрямую, поэтому удаляем и вставляем заново
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    
    _, err = tx.Exec("ALTER TABLE ipset_records DELETE WHERE id = ?", id)
    if err != nil {
        tx.Rollback()
        return err
    }
    
    _, err = tx.Exec(
        "INSERT INTO ipset_records (id, ip, cidr, port, protocol, description, context, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
        id, record.IP, record.CIDR, record.Port, record.Protocol, record.Description, record.Context, record.CreatedAt, record.UpdatedAt,
    )
    if err != nil {
        tx.Rollback()
        return err
    }
    
    return tx.Commit()
}

func (s *ClickHouseIPSetStorage) DeleteRecord(id int) error {
    _, err := s.db.Exec("ALTER TABLE ipset_records DELETE WHERE id = ?", id)
    return err
}

func (s *ClickHouseIPSetStorage) SearchRecords(context string) ([]*models.IPSetRecord, error) {
    // ClickHouse поддерживает полнотекстовый поиск через LIKE
    rows, err := s.db.Query(
        "SELECT id, ip, cidr, port, protocol, description, context, created_at, updated_at FROM ipset_records WHERE context LIKE ? OR description LIKE ?",
        "%"+context+"%", "%"+context+"%",
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
func (s *ClickHouseIPSetStorage) CreateIPSet(set *models.IPSet) error {
    set.CreatedAt = time.Now()
    set.UpdatedAt = time.Now()
    
    _, err := s.db.Exec(
        "INSERT INTO ipsets (name, type, family, hashsize, maxelem, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
        set.Name, set.Type, set.Family, set.HashSize, set.MaxElem, set.Description, set.CreatedAt, set.UpdatedAt,
    )
    return err
}

func (s *ClickHouseIPSetStorage) GetIPSet(name string) (*models.IPSet, error) {
    var set models.IPSet
    err := s.db.QueryRow(
        "SELECT name, type, family, hashsize, maxelem, description, created_at, updated_at FROM ipsets WHERE name = ?",
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

func (s *ClickHouseIPSetStorage) GetAllIPSets() ([]*models.IPSet, error) {
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

func (s *ClickHouseIPSetStorage) UpdateIPSet(name string, set *models.IPSet) error {
    set.UpdatedAt = time.Now()
    
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    
    _, err = tx.Exec("ALTER TABLE ipsets DELETE WHERE name = ?", name)
    if err != nil {
        tx.Rollback()
        return err
    }
    
    _, err = tx.Exec(
        "INSERT INTO ipsets (name, type, family, hashsize, maxelem, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
        set.Name, set.Type, set.Family, set.HashSize, set.MaxElem, set.Description, set.CreatedAt, set.UpdatedAt,
    )
    if err != nil {
        tx.Rollback()
        return err
    }
    
    return tx.Commit()
}

func (s *ClickHouseIPSetStorage) DeleteIPSet(name string) error {
    _, err := s.db.Exec("ALTER TABLE ipsets DELETE WHERE name = ?", name)
    return err
}

func (s *ClickHouseIPSetStorage) AddIPSetEntry(setName string, entry *models.IPSetEntry) error {
    entry.CreatedAt = time.Now()
    
    // Получаем следующий ID
    var maxID int
    err := s.db.QueryRow("SELECT max(id) FROM ipset_entries").Scan(&maxID)
    if err != nil {
        // Если таблица пустая, начинаем с 1
        maxID = 0
    }
    entry.ID = maxID + 1
    
    _, err = s.db.Exec(
        "INSERT INTO ipset_entries (id, ipset_name, value, comment, created_at) VALUES (?, ?, ?, ?, ?)",
        entry.ID, setName, entry.Value, entry.Comment, entry.CreatedAt,
    )
    return err
}

func (s *ClickHouseIPSetStorage) RemoveIPSetEntry(entryID int) error {
    _, err := s.db.Exec("ALTER TABLE ipset_entries DELETE WHERE id = ?", entryID)
    return err
}

func (s *ClickHouseIPSetStorage) GetIPSetEntries(setName string) ([]*models.IPSetEntry, error) {
    rows, err := s.db.Query("SELECT id, ipset_name, value, comment, created_at FROM ipset_entries WHERE ipset_name = ?", setName)
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

func (s *ClickHouseIPSetStorage) SearchIPSets(query string) ([]*models.IPSet, error) {
    rows, err := s.db.Query(
        "SELECT name, type, family, hashsize, maxelem, description, created_at, updated_at FROM ipsets WHERE name LIKE ? OR description LIKE ?",
        "%"+query+"%", "%"+query+"%",
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
func (s *ClickHouseIPSetStorage) CreateIPTablesRule(rule *models.IPTablesRule) error {
    rule.CreatedAt = time.Now()
    rule.UpdatedAt = time.Now()
    
    // Сериализуем массивы в JSON
    srcSetsJSON, _ := json.Marshal(rule.SrcSets)
    dstSetsJSON, _ := json.Marshal(rule.DstSets)
    
    _, err := s.db.Exec(
        "INSERT INTO iptables_rules (id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
        rule.ID, rule.Chain, rule.Interface, rule.Protocol, string(srcSetsJSON), string(dstSetsJSON), rule.Action, rule.Description, rule.Position, rule.CreatedAt, rule.UpdatedAt,
    )
    return err
}

func (s *ClickHouseIPSetStorage) GetIPTablesRule(id int) (*models.IPTablesRule, error) {
    var rule models.IPTablesRule
    var srcSetsJSON, dstSetsJSON string
    
    err := s.db.QueryRow(
        "SELECT id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at FROM iptables_rules WHERE id = ?",
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

func (s *ClickHouseIPSetStorage) GetAllIPTablesRules() ([]*models.IPTablesRule, error) {
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

func (s *ClickHouseIPSetStorage) UpdateIPTablesRule(id int, rule *models.IPTablesRule) error {
    rule.UpdatedAt = time.Now()
    
    srcSetsJSON, _ := json.Marshal(rule.SrcSets)
    dstSetsJSON, _ := json.Marshal(rule.DstSets)
    
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    
    _, err = tx.Exec("ALTER TABLE iptables_rules DELETE WHERE id = ?", id)
    if err != nil {
        tx.Rollback()
        return err
    }
    
    _, err = tx.Exec(
        "INSERT INTO iptables_rules (id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
        id, rule.Chain, rule.Interface, rule.Protocol, string(srcSetsJSON), string(dstSetsJSON), rule.Action, rule.Description, rule.Position, rule.CreatedAt, rule.UpdatedAt,
    )
    if err != nil {
        tx.Rollback()
        return err
    }
    
    return tx.Commit()
}

func (s *ClickHouseIPSetStorage) DeleteIPTablesRule(id int) error {
    _, err := s.db.Exec("ALTER TABLE iptables_rules DELETE WHERE id = ?", id)
    return err
}

func (s *ClickHouseIPSetStorage) ReorderIPTablesRule(id int, newPosition int) error {
    rule, err := s.GetIPTablesRule(id)
    if err != nil {
        return err
    }
    
    rule.Position = newPosition
    return s.UpdateIPTablesRule(id, rule)
}

func (s *ClickHouseIPSetStorage) SearchIPTablesRules(query string) ([]*models.IPTablesRule, error) {
    rows, err := s.db.Query(
        "SELECT id, chain, interface, protocol, src_sets, dst_sets, action, description, position, created_at, updated_at FROM iptables_rules WHERE chain LIKE ? OR description LIKE ?",
        "%"+query+"%", "%"+query+"%",
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