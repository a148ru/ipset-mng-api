package main

import (
    "fmt"
    "log"
    "time"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    "ipset-api-server/internal/storage"
    
    "github.com/google/uuid"
)

func main() {
    cfg := config.Load()
    
    keyStorage, err := storage.NewKeyStorage(cfg.AuthStorageType, cfg)
    if err != nil {
        log.Fatalf("Failed to initialize key storage: %v", err)
    }
    
    // Генерируем новый ключ
    key := &models.AuthKey{
        Key:       uuid.New().String(),
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().AddDate(1, 0, 0), // 1 год
        IsActive:  true,
    }
    
    if err := keyStorage.SaveKey(key); err != nil {
        log.Fatalf("Failed to save key: %v", err)
    }
    
    fmt.Printf("Generated API Key: %s\n", key.Key)
    fmt.Printf("Expires at: %s\n", key.ExpiresAt.Format(time.RFC3339))
}