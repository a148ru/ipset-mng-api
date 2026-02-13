// internal/storage/interface.go
package storage

import (
    "ipset-api/internal/models"
)

type KeyStorage interface {
    GetKey(key string) (*models.AuthKey, error)
    SaveKey(key *models.AuthKey) error
    DeleteKey(key string) error
    ListKeys() ([]*models.AuthKey, error)
}

type IPSetStorage interface {
    Create(record *models.IPSetRecord) error
    GetByID(id int) (*models.IPSetRecord, error)
    GetAll() ([]*models.IPSetRecord, error)
    Update(id int, record *models.IPSetRecord) error
    Delete(id int) error
    Search(context string) ([]*models.IPSetRecord, error)
}