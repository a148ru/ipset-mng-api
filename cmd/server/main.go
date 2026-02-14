package main

import (
    "fmt"
    "log"
    "ipset-api-server/internal/api"
    "ipset-api-server/internal/auth"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/storage"
    
    "github.com/joho/godotenv"
)

func main() {
    // Загружаем .env файл если существует
    godotenv.Load()

    // Загружаем конфигурацию
    cfg := config.Load()

    // Инициализируем хранилище авторизованных ключей
    authStorage, err := storage.NewKeyStorage(cfg.AuthStorageType, cfg)
    if err != nil {
        log.Fatalf("Failed to initialize auth storage: %v", err)
    }

    // Инициализируем хранилище ipset записей
    ipsetStorage, err := storage.NewIPSetStorage(cfg.IPSetStorageType, cfg)
    if err != nil {
        log.Fatalf("Failed to initialize ipset storage: %v", err)
    }

    // Инициализируем менеджер авторизации
    authManager := auth.NewManager(authStorage)

    // Инициализируем и запускаем API сервер
    server := api.NewServer(cfg, authManager, ipsetStorage)
    
    addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
    log.Printf("Server starting on %s", addr)
    
    if err := server.Run(addr); err != nil {
        log.Fatal(err)
    }
}

