package storage

import (
    "ipset-api-server/internal/models"
)

type KeyStorage interface {
    GetKey(key string) (*models.AuthKey, error)
    SaveKey(key *models.AuthKey) error
    DeleteKey(key string) error
    ListKeys() ([]*models.AuthKey, error)
}

type IPSetStorage interface {
    // Методы для IPSetRecord (существующие)
    CreateRecord(record *models.IPSetRecord) error
    GetRecordByID(id int) (*models.IPSetRecord, error)
    GetAllRecords() ([]*models.IPSetRecord, error)
    UpdateRecord(id int, record *models.IPSetRecord) error
    DeleteRecord(id int) error
    SearchRecords(context string) ([]*models.IPSetRecord, error)
    
    // Методы для управления ipset
    CreateIPSet(set *models.IPSet) error
    GetIPSet(name string) (*models.IPSet, error)
    GetAllIPSets() ([]*models.IPSet, error)
    UpdateIPSet(name string, set *models.IPSet) error
    DeleteIPSet(name string) error
    AddIPSetEntry(setName string, entry *models.IPSetEntry) error
    RemoveIPSetEntry(entryID int) error
    GetIPSetEntries(setName string) ([]*models.IPSetEntry, error)
    SearchIPSets(query string) ([]*models.IPSet, error)
    
    // Методы для управления iptables правилами
    CreateIPTablesRule(rule *models.IPTablesRule) error
    GetIPTablesRule(id int) (*models.IPTablesRule, error)
    GetAllIPTablesRules() ([]*models.IPTablesRule, error)
    UpdateIPTablesRule(id int, rule *models.IPTablesRule) error
    DeleteIPTablesRule(id int) error
    ReorderIPTablesRule(id int, newPosition int) error
    SearchIPTablesRules(query string) ([]*models.IPTablesRule, error)
}