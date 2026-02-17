package models

import (
    "time"
)

type AuthKey struct {
    Key       string    `json:"key"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
    IsActive  bool      `json:"is_active"`
}

type IPSetRecord struct {
    ID          int       `json:"id"`
    SetName     string    `json:"set_name"`
    IP          string    `json:"ip"`
    CIDR        string    `json:"cidr,omitempty"`
    Port        int       `json:"port,omitempty"`
    Protocol    string    `json:"protocol,omitempty"`
    Description string    `json:"description"`
    Context     string    `json:"context"`
    SetType     string    `json:"set_type,omitempty"`
    SetOptions  string    `json:"set_options,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type IPSetSet struct {
    Name        string         `json:"name"`
    Type        string         `json:"type"`
    Options     string         `json:"options,omitempty"`
    Records     []IPSetRecord  `json:"records"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
}

type ImportResult struct {
    SetName     string   `json:"set_name"`
    Records     int      `json:"records"`
    SetType     string   `json:"set_type"`
    Success     bool     `json:"success"`
    Error       string   `json:"error,omitempty"`
}

type LoginRequest struct {
    APIKey string `json:"api_key" binding:"required"`
}

type LoginResponse struct {
    Token string `json:"token"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

type SuccessResponse struct {
    Message string `json:"message"`
}

type IPSetType string
const (
    IPSetTypeHashNet     IPSetType = "hash:net"
    IPSetTypeHashIP      IPSetType = "hash:ip"
    IPSetTypeHashIPPort  IPSetType = "hash:ip,port"
    IPSetTypeHashNetPort IPSetType = "hash:net,port"
)

type IPSetFamily string
const (
    FamilyInet  IPSetFamily = "inet"
    FamilyInet6 IPSetFamily = "inet6"
)

type IPSet struct {
    Name        string         `json:"name" binding:"required"`
    Type        IPSetType      `json:"type" binding:"required"`
    Family      IPSetFamily    `json:"family" binding:"required"`
    HashSize    int            `json:"hashsize" binding:"required"`
    MaxElem     int            `json:"maxelem" binding:"required"`
    Entries     []IPSetEntry   `json:"entries"`
    Description string         `json:"description"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
}

type IPSetEntry struct {
    ID          int       `json:"id"`
    IPSetName   string    `json:"ipset_name"`
    Value       string    `json:"value"` // например: "192.168.0.0/16", "192.168.28.193,tcp:9644"
    Comment     string    `json:"comment"`
    CreatedAt   time.Time `json:"created_at"`
}

/* type CreateIPSetRequest struct {
    Name        string      `json:"name" binding:"required"`
    Type        IPSetType   `json:"type" binding:"required"`
    Family      IPSetFamily `json:"family" binding:"required"`
    HashSize    int         `json:"hashsize" binding:"required"`
    MaxElem     int         `json:"maxelem" binding:"required"`
    Description string      `json:"description"`
}
 */
type AddIPSetEntryRequest struct {
    Value   string `json:"value" binding:"required"`
    Comment string `json:"comment"`
}

type IPTablesRule struct {
    ID          int       `json:"id"`
    Chain       string    `json:"chain" binding:"required"`
    Interface   string    `json:"interface"`
    Protocol    string    `json:"protocol"`
    SrcSets     []string  `json:"src_sets"`      // имена ipset для source
    DstSets     []string  `json:"dst_sets"`      // имена ipset для destination
    Action      string    `json:"action" binding:"required"` // ACCEPT, DROP, REJECT
    Description string    `json:"description"`
    Position    int       `json:"position"`      // позиция в цепочке
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type CreateIPTablesRuleRequest struct {
    Chain       string   `json:"chain" binding:"required"`
    Interface   string   `json:"interface"`
    Protocol    string   `json:"protocol"`
    SrcSets     []string `json:"src_sets"`
    DstSets     []string `json:"dst_sets"`
    Action      string   `json:"action" binding:"required"`
    Description string   `json:"description"`
    Position    int      `json:"position"`
}

type IPSetConfig struct {
    Name       string `json:"name"`
    ConfigLine string `json:"config_line"` // полная строка конфигурации ipset
}

type IPSetCommand struct {
    Command string   `json:"command"` // create, add, del, flush, destroy
    SetName string   `json:"set_name"`
    Args    []string `json:"args"`
}

type ApplyRulesRequest struct {
    IPSetCommands    []IPSetCommand    `json:"ipset_commands"`
    IPTablesCommands []string          `json:"iptables_commands"` // команды iptables
}
// Отдельные структуры для запросов записей
type CreateIPSetRecordRequest struct {
    IP          string `json:"ip" binding:"required"`
    CIDR        string `json:"cidr"`
    Port        int    `json:"port"`
    Protocol    string `json:"protocol"`
    Description string `json:"description"`
    Context     string `json:"context" binding:"required"`
}

type UpdateIPSetRecordRequest struct {
    IP          string `json:"ip"`
    CIDR        string `json:"cidr"`
    Port        int    `json:"port"`
    Protocol    string `json:"protocol"`
    Description string `json:"description"`
    Context     string `json:"context"`
}

// Структуры для запросов ipset (уже были, но переименуем для ясности)
type CreateIPSetRequest struct {
    Name        string      `json:"name" binding:"required"`
    Type        IPSetType   `json:"type" binding:"required"`
    Family      IPSetFamily `json:"family" binding:"required"`
    HashSize    int         `json:"hashsize" binding:"required"`
    MaxElem     int         `json:"maxelem" binding:"required"`
    Description string      `json:"description"`
}

type UpdateIPSetRequest struct {
    Name        string      `json:"name"`
    Type        IPSetType   `json:"type"`
    Family      IPSetFamily `json:"family"`
    HashSize    int         `json:"hashsize"`
    MaxElem     int         `json:"maxelem"`
    Description string      `json:"description"`
}