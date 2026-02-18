-- docker/mysql/init.sql
CREATE DATABASE IF NOT EXISTS ipset;
USE ipset;

CREATE TABLE IF NOT EXISTS auth_keys (
    `key` VARCHAR(255) PRIMARY KEY,
    created_at DATETIME,
    expires_at DATETIME,
    is_active BOOLEAN
);

CREATE TABLE IF NOT EXISTS ipset_records (
    id INT PRIMARY KEY,
    ip VARCHAR(45),
    cidr VARCHAR(45),
    port INT,
    protocol VARCHAR(10),
    description TEXT,
    context TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS ipsets (
    name VARCHAR(255) PRIMARY KEY,
    type VARCHAR(50),
    family VARCHAR(10),
    hashsize INT,
    maxelem INT,
    description TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS ipset_entries (
    id INT AUTO_INCREMENT PRIMARY KEY,
    ipset_name VARCHAR(255),
    `value` TEXT,
    comment TEXT,
    created_at DATETIME,
    FOREIGN KEY (ipset_name) REFERENCES ipsets(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS iptables_rules (
    id INT PRIMARY KEY,
    chain VARCHAR(255),
    `interface` VARCHAR(255),
    protocol VARCHAR(10),
    src_sets JSON,
    dst_sets JSON,
    action VARCHAR(50),
    description TEXT,
    position INT,
    created_at DATETIME,
    updated_at DATETIME
);