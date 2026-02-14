package config

import (
    "os"
)

type Config struct {
    // Server settings
    ServerHost string
    ServerPort string
    
    // Auth settings
    AuthStorageType string
    JWTSecret       string
    
    // IPSet storage settings
    IPSetStorageType string
    
    // Database settings
    ClickHouseHost     string
    ClickHousePort     string
    ClickHouseDatabase string
    ClickHouseUsername string
    ClickHousePassword string
    
    MySQLHost     string
    MySQLPort     string
    MySQLDatabase string
    MySQLUsername string
    MySQLPassword string
    
    PostgreSQLHost     string
    PostgreSQLPort     string
    PostgreSQLDatabase string
    PostgreSQLUsername string
    PostgreSQLPassword string
    
    // File storage settings
    AuthKeysFilePath string
    IPSetFilePath    string
}

func Load() *Config {
    return &Config{
        ServerHost: getEnv("SERVER_HOST", "localhost"),
        ServerPort: getEnv("SERVER_PORT", "8080"),
        
        AuthStorageType: getEnv("AUTH_STORAGE_TYPE", "file"),
        JWTSecret:       getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
        
        IPSetStorageType: getEnv("IPSET_STORAGE_TYPE", "file"),
        
        ClickHouseHost:     getEnv("CLICKHOUSE_HOST", "localhost"),
        ClickHousePort:     getEnv("CLICKHOUSE_PORT", "9000"),
        ClickHouseDatabase: getEnv("CLICKHOUSE_DATABASE", "ipset"),
        ClickHouseUsername: getEnv("CLICKHOUSE_USERNAME", "default"),
        ClickHousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),
        
        MySQLHost:     getEnv("MYSQL_HOST", "localhost"),
        MySQLPort:     getEnv("MYSQL_PORT", "3306"),
        MySQLDatabase: getEnv("MYSQL_DATABASE", "ipset"),
        MySQLUsername: getEnv("MYSQL_USERNAME", "root"),
        MySQLPassword: getEnv("MYSQL_PASSWORD", ""),
        
        PostgreSQLHost:     getEnv("POSTGRES_HOST", "localhost"),
        PostgreSQLPort:     getEnv("POSTGRES_PORT", "5432"),
        PostgreSQLDatabase: getEnv("POSTGRES_DATABASE", "ipset"),
        PostgreSQLUsername: getEnv("POSTGRES_USERNAME", "postgres"),
        PostgreSQLPassword: getEnv("POSTGRES_PASSWORD", ""),
        
        AuthKeysFilePath: getEnv("AUTH_KEYS_FILE", "data/auth_keys.json"),
        IPSetFilePath:    getEnv("IPSET_FILE", "data/ipset_records.json"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

