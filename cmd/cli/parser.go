// cmd/cli/parser.go
package main

import (
    "fmt"
    "os"
    "os/exec"
    //"regexp"
    "strconv"
    "strings"
)

func parseIPSetFile(filename string) ([]ImportedRule, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    return parseIPSetData(string(data), filename)
}

func parseIPSetData(data string, source string) ([]ImportedRule, error) {
    var rules []ImportedRule
    lines := strings.Split(data, "\n")
    
    var currentSet struct {
        name    string
        setType string
        options string
    }
    
    for i, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        
        switch fields[0] {
        case "create":
            if len(fields) >= 3 {
                currentSet.name = fields[1]
                currentSet.setType = fields[2]
                if len(fields) > 3 {
                    currentSet.options = strings.Join(fields[3:], " ")
                }
            }
            
        case "add":
            if len(fields) >= 3 {
                setName := fields[1]
                entry := fields[2]
                
                setType := currentSet.setType
                if setName != currentSet.name {
                    if strings.Contains(setName, "tcp") {
                        setType = "hash:ip,port"
                    } else if strings.Contains(setName, "udp") {
                        setType = "hash:ip,port"
                    } else {
                        setType = "hash:ip"
                    }
                }
                
                rule := parseIPSetEntry(setName, entry, setType, currentSet.options)
                if rule != nil {
                    rule.LineNumber = i + 1
                    rule.Description = fmt.Sprintf("Imported from %s line %d", source, i+1)
                    rule.Context = fmt.Sprintf("%s:%s", source, setName)
                    rules = append(rules, *rule)
                }
            }
        }
    }
    
    return rules, nil
}

func parseIPSetEntry(setName, entry, setType, setOptions string) *ImportedRule {
    rule := &ImportedRule{
        SetName:    setName,
        SetType:    setType,
        SetOptions: setOptions,
    }
    
    // Парсим entry
    if strings.Contains(entry, ",") {
        parts := strings.Split(entry, ",")
        ipPart := parts[0]
        
        // Парсим IP часть
        if strings.Contains(ipPart, "/") {
            ipCIDR := strings.Split(ipPart, "/")
            rule.IP = ipCIDR[0]
            rule.CIDR = ipCIDR[1]
        } else if strings.Contains(ipPart, ":") {
            ipPort := strings.Split(ipPart, ":")
            rule.IP = ipPort[0]
            if port, err := strconv.Atoi(ipPort[1]); err == nil {
                rule.Port = port
            }
        } else {
            rule.IP = ipPart
        }
        
        // Парсим proto:port часть
        if len(parts) > 1 {
            protoPort := parts[1]
            if strings.Contains(protoPort, ":") {
                pp := strings.Split(protoPort, ":")
                rule.Protocol = pp[0]
                if len(pp) > 1 {
                    if port, err := strconv.Atoi(pp[1]); err == nil {
                        rule.Port = port
                    }
                }
            }
        }
    } else {
        rule.IP = entry
        if strings.Contains(entry, "/") {
            ipCIDR := strings.Split(entry, "/")
            rule.IP = ipCIDR[0]
            rule.CIDR = ipCIDR[1]
        }
    }
    
    return rule
}

func getIPSetSets() ([]string, error) {
    cmd := exec.Command("ipset", "list", "-n")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    var sets []string
    for _, line := range strings.Split(string(output), "\n") {
        line = strings.TrimSpace(line)
        if line != "" {
            sets = append(sets, line)
        }
    }
    
    return sets, nil
}

func getIPSetRules(setName string) ([]ImportedRule, error) {
    cmd := exec.Command("ipset", "save", setName)
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    return parseIPSetData(string(output), fmt.Sprintf("system:%s", setName))
}

func commandExists(cmd string) bool {
    _, err := exec.LookPath(cmd)
    return err == nil
}