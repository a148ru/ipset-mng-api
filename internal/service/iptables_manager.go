package service

import (
    "fmt"
    "os/exec"
    "strings"
    "ipset-api-server/internal/models"
)

type IPTablesManager struct {
    dryRun bool
}

func NewIPTablesManager(dryRun bool) *IPTablesManager {
    return &IPTablesManager{
        dryRun: dryRun,
    }
}

// GenerateIPSetCommand генерирует команду для создания ipset
func (m *IPTablesManager) GenerateIPSetCommand(set *models.IPSet) string {
    cmd := fmt.Sprintf("create %s %s family %s hashsize %d maxelem %d",
        set.Name, set.Type, set.Family, set.HashSize, set.MaxElem)
    return cmd
}

// GenerateIPSetAddCommand генерирует команду для добавления элемента в ipset
func (m *IPTablesManager) GenerateIPSetAddCommand(setName string, entry *models.IPSetEntry) string {
    return fmt.Sprintf("add %s %s", setName, entry.Value)
}

// GenerateIPSetDeleteCommand генерирует команду для удаления элемента из ipset
func (m *IPTablesManager) GenerateIPSetDeleteCommand(setName string, entry *models.IPSetEntry) string {
    return fmt.Sprintf("del %s %s", setName, entry.Value)
}

// GenerateIPTablesRule генерирует команду iptables из правила
func (m *IPTablesManager) GenerateIPTablesRule(rule *models.IPTablesRule, operation string) string {
    parts := []string{"iptables"}
    
    if operation == "insert" {
        parts = append(parts, "-I")
    } else {
        parts = append(parts, "-A")
    }
    
    parts = append(parts, rule.Chain)
    
    if rule.Interface != "" {
        parts = append(parts, "-i", rule.Interface)
    }
    
    if rule.Protocol != "" {
        parts = append(parts, "-p", rule.Protocol)
    }
    
    // Добавляем source sets
    for _, set := range rule.SrcSets {
        parts = append(parts, "-m", "set", "--match-set", set, "src")
    }
    
    // Добавляем destination sets
    for _, set := range rule.DstSets {
        parts = append(parts, "-m", "set", "--match-set", set, "dst,dst")
    }
    
    parts = append(parts, "-j", rule.Action)
    
    return strings.Join(parts, " ")
}

// ApplyConfiguration применяет конфигурацию ipset и iptables
func (m *IPTablesManager) ApplyConfiguration(ipsetCommands []models.IPSetCommand, iptablesCommands []string) error {
    if m.dryRun {
        fmt.Println("=== DRY RUN MODE - Commands that would be executed ===")
        for _, cmd := range ipsetCommands {
            fmt.Printf("ipset %s %s %s\n", cmd.Command, cmd.SetName, strings.Join(cmd.Args, " "))
        }
        for _, cmd := range iptablesCommands {
            fmt.Println(cmd)
        }
        return nil
    }
    
    // Применяем ipset команды
    for _, cmd := range ipsetCommands {
        args := append([]string{cmd.Command, cmd.SetName}, cmd.Args...)
        if err := m.execCommand("ipset", args...); err != nil {
            return fmt.Errorf("failed to execute ipset command: %v", err)
        }
    }
    
    // Применяем iptables команды
    for _, cmd := range iptablesCommands {
        args := strings.Fields(cmd)
        if len(args) == 0 {
            continue
        }
        if err := m.execCommand(args[0], args[1:]...); err != nil {
            return fmt.Errorf("failed to execute iptables command: %v", err)
        }
    }
    
    return nil
}

func (m *IPTablesManager) execCommand(name string, arg ...string) error {
    cmd := exec.Command(name, arg...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("command failed: %s\nOutput: %s", err, string(output))
    }
    return nil
}

// ParseIPSetConfig парсит конфигурацию ipset из строки
func (m *IPTablesManager) ParseIPSetConfig(config string) (*models.IPSet, error) {
    parts := strings.Fields(config)
    if len(parts) < 6 || parts[0] != "create" {
        return nil, fmt.Errorf("invalid ipset config format")
    }
    
    set := &models.IPSet{
        Name:     parts[1],
        Type:     models.IPSetType(parts[2]),
        Family:   models.IPSetFamily(parts[4]),
        HashSize: 1024, // default
        MaxElem:  65536, // default
    }
    
    // Parse optional parameters
    for i := 5; i < len(parts); i++ {
        switch parts[i] {
        case "hashsize":
            if i+1 < len(parts) {
                fmt.Sscanf(parts[i+1], "%d", &set.HashSize)
                i++
            }
        case "maxelem":
            if i+1 < len(parts) {
                fmt.Sscanf(parts[i+1], "%d", &set.MaxElem)
                i++
            }
        }
    }
    
    return set, nil
}

// ParseIPSetEntry парсит запись ipset из строки
func (m *IPTablesManager) ParseIPSetEntry(line string) (string, string, error) {
    parts := strings.Fields(line)
    if len(parts) < 3 || parts[0] != "add" {
        return "", "", fmt.Errorf("invalid ipset entry format")
    }
    
    return parts[1], parts[2], nil
}