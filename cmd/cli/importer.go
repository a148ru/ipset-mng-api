// cmd/cli/importer.go
package main

import (
    "encoding/json"
    "fmt"
    //"strings"
)

func importRules(rules []ImportedRule, contextPrefix string, dryRun bool) {
    if len(rules) == 0 {
        fmt.Println("No rules to import")
        return
    }
    
    // Группируем правила по set_name
    setMap := make(map[string][]ImportedRule)
    for _, rule := range rules {
        setMap[rule.SetName] = append(setMap[rule.SetName], rule)
    }
    
    fmt.Printf("Found %d rules in %d sets to import\n", len(rules), len(setMap))
    
    if dryRun {
        printDryRun(setMap)
        return
    }
    
    // Реальный импорт
    if config.Token == "" {
        fmt.Println("Error: Not authenticated. Please login first using 'ipset-cli login'")
        return
    }
    
    performImport(setMap, contextPrefix)
}

func printDryRun(setMap map[string][]ImportedRule) {
    fmt.Println("\nDRY RUN - Sets that would be imported:")
    for setName, setRules := range setMap {
        fmt.Printf("\n=== Set: %s ===\n", setName)
        if len(setRules) > 0 {
            fmt.Printf("  Type: %s\n", setRules[0].SetType)
            if setRules[0].SetOptions != "" {
                fmt.Printf("  Options: %s\n", setRules[0].SetOptions)
            }
        }
        fmt.Printf("  Rules: %d\n", len(setRules))
        
        for i, rule := range setRules {
            fmt.Printf("    %d. %s", i+1, rule.IP)
            if rule.CIDR != "" {
                fmt.Printf("/%s", rule.CIDR)
            }
            if rule.Port != 0 {
                fmt.Printf(":%d", rule.Port)
            }
            if rule.Protocol != "" {
                fmt.Printf(" (%s)", rule.Protocol)
            }
            fmt.Println()
        }
    }
}

func performImport(setMap map[string][]ImportedRule, contextPrefix string) {
    var totalSuccess, totalFailed int
    
    for setName, setRules := range setMap {
        fmt.Printf("\nImporting set: %s (%d rules)\n", setName, len(setRules))
        
        // Подготавливаем данные для импорта сета
        importData := map[string]interface{}{
            "set_name":    setName,
            "context":     fmt.Sprintf("%s:%s", contextPrefix, setName),
            "description": fmt.Sprintf("Imported set %s", setName),
            "records":     []map[string]interface{}{},
        }
        
        if len(setRules) > 0 {
            importData["set_type"] = setRules[0].SetType
            if setRules[0].SetOptions != "" {
                importData["set_options"] = setRules[0].SetOptions
            }
        }
        
        // Создаем сет через API
        jsonData, _ := json.Marshal(importData)
        _, err := makeRequestWithBody("POST", "/sets/import", jsonData)
        if err != nil {
            fmt.Printf("❌ Failed to create set %s: %v\n", setName, err)
            totalFailed += len(setRules)
            continue
        }
        
        // Импортируем правила
        var successCount, failCount int
        
        for _, rule := range setRules {
            record := map[string]interface{}{
                "set_name":    rule.SetName,
                "ip":          rule.IP,
                "context":     fmt.Sprintf("%s:%s", contextPrefix, rule.Context),
                "description": rule.Description,
                "set_type":    rule.SetType,
            }
            
            if rule.SetOptions != "" {
                record["set_options"] = rule.SetOptions
            }
            if rule.CIDR != "" {
                record["cidr"] = rule.CIDR
            }
            if rule.Port != 0 {
                record["port"] = rule.Port
            }
            if rule.Protocol != "" {
                record["protocol"] = rule.Protocol
            }
            
            jsonData, _ := json.Marshal(record)
            _, err := makeRequestWithBody("POST", "/records", jsonData)
            
            if err != nil {
                fmt.Printf("  ❌ Failed: %s", rule.IP)
                if rule.Port != 0 {
                    fmt.Printf(":%d", rule.Port)
                }
                fmt.Printf(" - %v\n", err)
                failCount++
            } else {
                fmt.Printf("  ✅ %s", rule.IP)
                if rule.Port != 0 {
                    fmt.Printf(":%d", rule.Port)
                }
                fmt.Println()
                successCount++
            }
        }
        
        fmt.Printf("  Set %s: %d successful, %d failed\n", setName, successCount, failCount)
        totalSuccess += successCount
        totalFailed += failCount
    }
    
    fmt.Printf("\nImport completed: %d successful, %d failed across %d sets\n", 
        totalSuccess, totalFailed, len(setMap))
}