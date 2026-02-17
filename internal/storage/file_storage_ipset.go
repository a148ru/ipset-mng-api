package storage

import (

    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"


    "ipset-api-server/internal/models"
)

// FileIPSetStorage объединяет все методы для работы с ipset и iptables
type FileIPSetStorage struct {
    filePath     string
    mu           sync.RWMutex
    nextID       int
    nextEntryID  int
}

func NewFileIPSetStorage(filePath string) (*FileIPSetStorage, error) {
    if err := os.MkdirAll("data", 0755); err != nil {
        return nil, err
    }
    
    storage := &FileIPSetStorage{
        filePath:    filePath,
        nextID:      100000,
        nextEntryID: 1,
    }
    
    // Создаем файл если не существует
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        initialData := &FileIPSetData{ // Исправлено: используем указатель
            IPSets:      make(map[string]*models.IPSet),
            IPTables:    make(map[int]*models.IPTablesRule),
            Records:     make(map[int]*models.IPSetRecord),
            NextID:      100000,
            NextEntryID: 1,
        }
        if err := storage.writeData(initialData); err != nil {
            return nil, err
        }
    }
    
    return storage, nil
}

type FileIPSetData struct {
    IPSets      map[string]*models.IPSet          `json:"ipsets"`
    IPTables    map[int]*models.IPTablesRule      `json:"iptables"`
    Records     map[int]*models.IPSetRecord       `json:"records"`
    NextID      int                                `json:"next_id"`
    NextEntryID int                                `json:"next_entry_id"`
}

func (s *FileIPSetStorage) readData() (*FileIPSetData, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    data, err := os.ReadFile(s.filePath)
    if err != nil {
        return nil, err
    }
    
    var fileData FileIPSetData
    if err := json.Unmarshal(data, &fileData); err != nil {
        return nil, err
    }
    
    return &fileData, nil
}

func (s *FileIPSetStorage) writeData(fileData *FileIPSetData) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    data, err := json.MarshalIndent(fileData, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(s.filePath, data, 0644)
}

// IPSet management methods
func (s *FileIPSetStorage) CreateIPSet(set *models.IPSet) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    if _, exists := fileData.IPSets[set.Name]; exists {
        return fmt.Errorf("ipset with name %s already exists", set.Name)
    }
    
    set.CreatedAt = time.Now()
    set.UpdatedAt = time.Now()
    set.Entries = []models.IPSetEntry{}
    
    fileData.IPSets[set.Name] = set
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) GetIPSet(name string) (*models.IPSet, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    set, exists := fileData.IPSets[name]
    if !exists {
        return nil, fmt.Errorf("ipset %s not found", name)
    }
    
    return set, nil
}

func (s *FileIPSetStorage) GetAllIPSets() ([]*models.IPSet, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    sets := make([]*models.IPSet, 0, len(fileData.IPSets))
    for _, set := range fileData.IPSets {
        sets = append(sets, set)
    }
    
    return sets, nil
}

func (s *FileIPSetStorage) UpdateIPSet(name string, set *models.IPSet) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    existingSet, exists := fileData.IPSets[name]
    if !exists {
        return fmt.Errorf("ipset %s not found", name)
    }
    
    // Сохраняем существующие entries
    set.Entries = existingSet.Entries
    set.CreatedAt = existingSet.CreatedAt
    set.UpdatedAt = time.Now()
    
    // Если имя изменилось, удаляем старую запись
    if name != set.Name {
        delete(fileData.IPSets, name)
    }
    
    fileData.IPSets[set.Name] = set
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) DeleteIPSet(name string) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    if _, exists := fileData.IPSets[name]; !exists {
        return fmt.Errorf("ipset %s not found", name)
    }
    
    delete(fileData.IPSets, name)
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) AddIPSetEntry(setName string, entry *models.IPSetEntry) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    set, exists := fileData.IPSets[setName]
    if !exists {
        return fmt.Errorf("ipset %s not found", setName)
    }
    
    entry.ID = fileData.NextEntryID
    entry.IPSetName = setName
    entry.CreatedAt = time.Now()
    
    fileData.NextEntryID++
    set.Entries = append(set.Entries, *entry)
    set.UpdatedAt = time.Now()
    
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) RemoveIPSetEntry(entryID int) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    for _, set := range fileData.IPSets {
        for i, entry := range set.Entries {
            if entry.ID == entryID {
                set.Entries = append(set.Entries[:i], set.Entries[i+1:]...)
                set.UpdatedAt = time.Now()
                return s.writeData(fileData)
            }
        }
    }
    
    return fmt.Errorf("entry with id %d not found", entryID)
}

func (s *FileIPSetStorage) GetIPSetEntries(setName string) ([]*models.IPSetEntry, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    set, exists := fileData.IPSets[setName]
    if !exists {
        return nil, fmt.Errorf("ipset %s not found", setName)
    }
    
    entries := make([]*models.IPSetEntry, len(set.Entries))
    for i := range set.Entries {
        entries[i] = &set.Entries[i]
    }
    
    return entries, nil
}

func (s *FileIPSetStorage) SearchIPSets(query string) ([]*models.IPSet, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    var results []*models.IPSet
    for _, set := range fileData.IPSets {
        if contains(set.Name, query) || contains(set.Description, query) {
            results = append(results, set)
        }
    }
    
    return results, nil
}

// IPTables rules management
func (s *FileIPSetStorage) CreateIPTablesRule(rule *models.IPTablesRule) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    rule.ID = fileData.NextID
    rule.CreatedAt = time.Now()
    rule.UpdatedAt = time.Now()
    
    fileData.NextID++
    fileData.IPTables[rule.ID] = rule
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) GetIPTablesRule(id int) (*models.IPTablesRule, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    rule, exists := fileData.IPTables[id]
    if !exists {
        return nil, fmt.Errorf("rule with id %d not found", id)
    }
    
    return rule, nil
}

func (s *FileIPSetStorage) GetAllIPTablesRules() ([]*models.IPTablesRule, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    rules := make([]*models.IPTablesRule, 0, len(fileData.IPTables))
    for _, rule := range fileData.IPTables {
        rules = append(rules, rule)
    }
    
    return rules, nil
}

func (s *FileIPSetStorage) UpdateIPTablesRule(id int, rule *models.IPTablesRule) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    existingRule, exists := fileData.IPTables[id]
    if !exists {
        return fmt.Errorf("rule with id %d not found", id)
    }
    
    rule.ID = id
    rule.UpdatedAt = time.Now()
    rule.CreatedAt = existingRule.CreatedAt
    
    fileData.IPTables[id] = rule
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) DeleteIPTablesRule(id int) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    if _, exists := fileData.IPTables[id]; !exists {
        return fmt.Errorf("rule with id %d not found", id)
    }
    
    delete(fileData.IPTables, id)
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) ReorderIPTablesRule(id int, newPosition int) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    rule, exists := fileData.IPTables[id]
    if !exists {
        return fmt.Errorf("rule with id %d not found", id)
    }
    
    rule.Position = newPosition
    rule.UpdatedAt = time.Now()
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) SearchIPTablesRules(query string) ([]*models.IPTablesRule, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    var results []*models.IPTablesRule
    for _, rule := range fileData.IPTables {
        if contains(rule.Chain, query) || contains(rule.Description, query) {
            results = append(results, rule)
        }
    }
    
    return results, nil
}

// IPSetRecord management methods (для совместимости с интерфейсом)
func (s *FileIPSetStorage) CreateRecord(record *models.IPSetRecord) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    // Генерируем 6-значный ID
    for fileData.NextID < 1000000 {
        if _, exists := fileData.Records[fileData.NextID]; !exists {
            record.ID = fileData.NextID
            fileData.NextID++
            break
        }
        fileData.NextID++
    }
    
    record.CreatedAt = time.Now()
    record.UpdatedAt = time.Now()
    fileData.Records[record.ID] = record
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) GetRecordByID(id int) (*models.IPSetRecord, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    record, exists := fileData.Records[id]
    if !exists {
        return nil, fmt.Errorf("record with id %d not found", id)
    }
    
    return record, nil
}

func (s *FileIPSetStorage) GetAllRecords() ([]*models.IPSetRecord, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    records := make([]*models.IPSetRecord, 0, len(fileData.Records))
    for _, record := range fileData.Records {
        records = append(records, record)
    }
    
    return records, nil
}

func (s *FileIPSetStorage) UpdateRecord(id int, record *models.IPSetRecord) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    existingRecord, exists := fileData.Records[id]
    if !exists {
        return fmt.Errorf("record with id %d not found", id)
    }
    
    record.ID = id
    record.UpdatedAt = time.Now()
    record.CreatedAt = existingRecord.CreatedAt
    fileData.Records[id] = record
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) DeleteRecord(id int) error {
    fileData, err := s.readData()
    if err != nil {
        return err
    }
    
    if _, exists := fileData.Records[id]; !exists {
        return fmt.Errorf("record with id %d not found", id)
    }
    
    delete(fileData.Records, id)
    return s.writeData(fileData)
}

func (s *FileIPSetStorage) SearchRecords(context string) ([]*models.IPSetRecord, error) {
    fileData, err := s.readData()
    if err != nil {
        return nil, err
    }
    
    var results []*models.IPSetRecord
    for _, record := range fileData.Records {
        if contains(record.Context, context) || contains(record.Description, context) {
            results = append(results, record)
        }
    }
    
    return results, nil
}