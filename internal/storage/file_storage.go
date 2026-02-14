package storage

import (
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"
    "ipset-api-server/internal/models"
    "strings"
)
// FileKeyStorage - реализация для хранения ключей в файле
type FileKeyStorage struct {
    filePath string
    mu       sync.RWMutex
}

// FileIPSetStorage - реализация для хранения ipset записей в файле
type FileIPSetStorage struct {
    filePath string
    mu       sync.RWMutex
    nextID   int
}

func NewFileKeyStorage(filePath string) (*FileKeyStorage, error) {
    // Создаем директорию если не существует
    if err := os.MkdirAll("data", 0755); err != nil {
        return nil, err
    }
    
    // Создаем файл если не существует
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
            return nil, err
        }
    }
    
    return &FileKeyStorage{
        filePath: filePath,
    }, nil
}

func (s *FileKeyStorage) readKeys() (map[string]*models.AuthKey, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    data, err := os.ReadFile(s.filePath)
    if err != nil {
        return nil, err
    }
    
    var keys map[string]*models.AuthKey
    if err := json.Unmarshal(data, &keys); err != nil {
        return nil, err
    }
    
    return keys, nil
}

func (s *FileKeyStorage) writeKeys(keys map[string]*models.AuthKey) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    data, err := json.MarshalIndent(keys, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(s.filePath, data, 0644)
}

func (s *FileKeyStorage) GetKey(key string) (*models.AuthKey, error) {
    keys, err := s.readKeys()
    if err != nil {
        return nil, err
    }
    
    return keys[key], nil
}

func (s *FileKeyStorage) SaveKey(key *models.AuthKey) error {
    keys, err := s.readKeys()
    if err != nil {
        return err
    }
    
    keys[key.Key] = key
    return s.writeKeys(keys)
}

func (s *FileKeyStorage) DeleteKey(key string) error {
    keys, err := s.readKeys()
    if err != nil {
        return err
    }
    
    delete(keys, key)
    return s.writeKeys(keys)
}

func (s *FileKeyStorage) ListKeys() ([]*models.AuthKey, error) {
    keys, err := s.readKeys()
    if err != nil {
        return nil, err
    }
    
    result := make([]*models.AuthKey, 0, len(keys))
    for _, key := range keys {
        result = append(result, key)
    }
    
    return result, nil
}

func NewFileIPSetStorage(filePath string) (*FileIPSetStorage, error) {
    if err := os.MkdirAll("data", 0755); err != nil {
        return nil, err
    }
    
    storage := &FileIPSetStorage{
        filePath: filePath,
        nextID:   100000, // Начинаем с 6-значных чисел
    }
    
    // Создаем файл если не существует
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        if err := storage.writeRecords(map[int]*models.IPSetRecord{}); err != nil {
            return nil, err
        }
    }
    
    return storage, nil
}

func (s *FileIPSetStorage) readRecords() (map[int]*models.IPSetRecord, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    data, err := os.ReadFile(s.filePath)
    if err != nil {
        return nil, err
    }
    
    var records map[int]*models.IPSetRecord
    if err := json.Unmarshal(data, &records); err != nil {
        return nil, err
    }
    
    return records, nil
}

func (s *FileIPSetStorage) writeRecords(records map[int]*models.IPSetRecord) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    data, err := json.MarshalIndent(records, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(s.filePath, data, 0644)
}

func (s *FileIPSetStorage) Create(record *models.IPSetRecord) error {
    records, err := s.readRecords()
    if err != nil {
        return err
    }
    
    // Генерируем 6-значный ID
    for s.nextID < 1000000 {
        if _, exists := records[s.nextID]; !exists {
            record.ID = s.nextID
            s.nextID++
            break
        }
        s.nextID++
    }
    
    now := time.Now()
    record.CreatedAt = now
    record.UpdatedAt = now
    records[record.ID] = record
    
    return s.writeRecords(records)
}

func (s *FileIPSetStorage) GetByID(id int) (*models.IPSetRecord, error) {
    records, err := s.readRecords()
    if err != nil {
        return nil, err
    }
    
    record, exists := records[id]
    if !exists {
        return nil, fmt.Errorf("record with id %d not found", id)
    }
    
    return record, nil
}

func (s *FileIPSetStorage) GetAll() ([]*models.IPSetRecord, error) {
    records, err := s.readRecords()
    if err != nil {
        return nil, err
    }
    
    result := make([]*models.IPSetRecord, 0, len(records))
    for _, record := range records {
        result = append(result, record)
    }
    
    return result, nil
}

func (s *FileIPSetStorage) GetBySetName(setName string) ([]*models.IPSetRecord, error) {
    records, err := s.readRecords()
    if err != nil {
        return nil, err
    }
    
    var result []*models.IPSetRecord
    for _, record := range records {
        if record.SetName == setName {
            result = append(result, record)
        }
    }
    
    if len(result) == 0 {
        return nil, fmt.Errorf("set %s not found", setName)
    }
    
    return result, nil
}

func (s *FileIPSetStorage) GetAllSets() ([]*models.IPSetSet, error) {
    records, err := s.readRecords()
    if err != nil {
        return nil, err
    }
    
    setMap := make(map[string]*models.IPSetSet)
    
    for _, record := range records {
        if set, exists := setMap[record.SetName]; exists {
            set.Records = append(set.Records, *record)
            if record.UpdatedAt.After(set.UpdatedAt) {
                set.UpdatedAt = record.UpdatedAt
            }
        } else {
            setMap[record.SetName] = &models.IPSetSet{
                Name:      record.SetName,
                Type:      record.SetType,
                Options:   record.SetOptions,
                Records:   []models.IPSetRecord{*record},
                CreatedAt: record.CreatedAt,
                UpdatedAt: record.UpdatedAt,
            }
        }
    }
    
    result := make([]*models.IPSetSet, 0, len(setMap))
    for _, set := range setMap {
        result = append(result, set)
    }
    
    return result, nil
}

func (s *FileIPSetStorage) Update(id int, record *models.IPSetRecord) error {
    records, err := s.readRecords()
    if err != nil {
        return err
    }
    
    existing, exists := records[id]
    if !exists {
        return fmt.Errorf("record with id %d not found", id)
    }
    
    record.ID = id
    record.CreatedAt = existing.CreatedAt
    record.UpdatedAt = time.Now()
    records[id] = record
    
    return s.writeRecords(records)
}

func (s *FileIPSetStorage) Delete(id int) error {
    records, err := s.readRecords()
    if err != nil {
        return err
    }
    
    if _, exists := records[id]; !exists {
        return fmt.Errorf("record with id %d not found", id)
    }
    
    delete(records, id)
    return s.writeRecords(records)
}

func (s *FileIPSetStorage) DeleteSet(setName string) error {
    records, err := s.readRecords()
    if err != nil {
        return err
    }
    
    found := false
    for id, record := range records {
        if record.SetName == setName {
            delete(records, id)
            found = true
        }
    }
    
    if !found {
        return fmt.Errorf("set %s not found", setName)
    }
    
    return s.writeRecords(records)
}

func (s *FileIPSetStorage) Search(query string) ([]*models.IPSetRecord, error) {
    records, err := s.readRecords()
    if err != nil {
        return nil, err
    }
    
    var result []*models.IPSetRecord
    query = strings.ToLower(query)
    
    for _, record := range records {
        if strings.Contains(strings.ToLower(record.Context), query) ||
           strings.Contains(strings.ToLower(record.Description), query) ||
           strings.Contains(strings.ToLower(record.IP), query) ||
           strings.Contains(strings.ToLower(record.SetName), query) {
            result = append(result, record)
        }
    }
    
    return result, nil
}
