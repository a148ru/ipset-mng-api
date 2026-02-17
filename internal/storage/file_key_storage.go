// internal/storage/file_key_storage.go
package storage

import (
    "encoding/json"
    "os"
    "sync"
    "ipset-api-server/internal/models"
)

type FileKeyStorage struct {
    filePath string
    mu       sync.RWMutex
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