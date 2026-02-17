// internal/storage/factory.go (исправленный)
package storage

import (
    "fmt"
    "ipset-api-server/internal/config"
)

func NewKeyStorage(storageType string, cfg *config.Config) (KeyStorage, error) {
    switch storageType {
    case "file":
        return NewFileKeyStorage(cfg.AuthKeysFilePath)
    case "mysql":
        return NewMySQLKeyStorage(cfg)
    case "postgresql":
        return NewPostgreSQLKeyStorage(cfg)
    case "clickhouse":
        return NewClickHouseKeyStorage(cfg)
    default:
        return nil, fmt.Errorf("unsupported storage type: %s", storageType)
    }
}

func NewIPSetStorage(storageType string, cfg *config.Config) (IPSetStorage, error) {
    switch storageType {
    case "file":
        return NewFileIPSetStorage(cfg.IPSetFilePath)
    case "mysql":
        return NewMySQLIPSetStorage(cfg)
    case "postgresql":
        return NewPostgreSQLIPSetStorage(cfg)
    case "clickhouse":
        return NewClickHouseIPSetStorage(cfg)
    default:
        return nil, fmt.Errorf("unsupported storage type: %s", storageType)
    }
}