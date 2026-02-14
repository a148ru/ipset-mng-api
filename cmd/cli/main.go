// cmd/cli/main.go (дополнение к существующему файлу)
package main

import (
//    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
//    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "time"
    "os/exec"

    "gopkg.in/yaml.v3"
    "github.com/olekukonko/tablewriter"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

type Config struct {
    APIURL   string `mapstructure:"api_url"`
    Token    string `mapstructure:"token"`
    Output   string `mapstructure:"output"`
    Insecure bool   `mapstructure:"insecure"`
}

type ImportedRule struct {
    IP          string
    CIDR        string
    Port        int
    Protocol    string
    Description string
    Context     string
    SetName     string
    LineNumber  int
}

var config Config
var client *http.Client

// Вспомогательная функция для выполнения команд
func execCommand(name string, args ...string) *exec.Cmd {
    cmd := exec.Command(name, args...)
    return cmd
}

func main() {
    initConfig()
    
    client = &http.Client{
        Timeout: 10 * time.Second,
    }
    
    rootCmd := &cobra.Command{
        Use:   "ipset-cli",
        Short: "CLI for IPSet API management",
        Long:  `A command line tool to manage IPSet records through REST API`,
        PersistentPreRun: func(cmd *cobra.Command, args []string) {
            viper.Unmarshal(&config)
        },
    }

    rootCmd.PersistentFlags().StringVar(&config.APIURL, "api-url", "http://localhost:8080", "API URL")
    rootCmd.PersistentFlags().StringVar(&config.Token, "token", "", "Authentication token")
    rootCmd.PersistentFlags().StringVar(&config.Output, "output", "table", "Output format (json, yaml, table, ipset)")
    rootCmd.PersistentFlags().BoolVar(&config.Insecure, "insecure", false, "Skip TLS verification")
    
    viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
    viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
    viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
    viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))

    // Login command
    loginCmd := &cobra.Command{
        Use:   "login [api-key]",
        Short: "Login and get JWT token",
        Args:  cobra.ExactArgs(1),
        Run:   runLogin,
    }
    rootCmd.AddCommand(loginCmd)

    // Records commands
    recordsCmd := &cobra.Command{
        Use:   "records",
        Short: "Manage IPSet records",
    }

    recordsCmd.AddCommand(&cobra.Command{
        Use:   "list",
        Short: "List all records",
        Run:   runListRecords,
    })

    recordsCmd.AddCommand(&cobra.Command{
        Use:   "get [id]",
        Short: "Get record by ID",
        Args:  cobra.ExactArgs(1),
        Run:   runGetRecord,
    })

    createCmd := &cobra.Command{
        Use:   "create",
        Short: "Create a new record",
        Run:   runCreateRecord,
    }
    createCmd.Flags().StringP("ip", "i", "", "IP address (required)")
    createCmd.Flags().StringP("cidr", "c", "", "CIDR mask")
    createCmd.Flags().IntP("port", "p", 0, "Port number")
    createCmd.Flags().StringP("protocol", "r", "", "Protocol (tcp/udp)")
    createCmd.Flags().StringP("description", "d", "", "Description")
    createCmd.Flags().StringP("context", "x", "", "Context (required)")
    createCmd.MarkFlagRequired("ip")
    createCmd.MarkFlagRequired("context")
    recordsCmd.AddCommand(createCmd)

    updateCmd := &cobra.Command{
        Use:   "update [id]",
        Short: "Update an existing record",
        Args:  cobra.ExactArgs(1),
        Run:   runUpdateRecord,
    }
    updateCmd.Flags().StringP("ip", "i", "", "IP address")
    updateCmd.Flags().StringP("cidr", "c", "", "CIDR mask")
    updateCmd.Flags().IntP("port", "p", 0, "Port number")
    updateCmd.Flags().StringP("protocol", "r", "", "Protocol (tcp/udp)")
    updateCmd.Flags().StringP("description", "d", "", "Description")
    updateCmd.Flags().StringP("context", "x", "", "Context")
    recordsCmd.AddCommand(updateCmd)

    recordsCmd.AddCommand(&cobra.Command{
        Use:   "delete [id]",
        Short: "Delete a record",
        Args:  cobra.ExactArgs(1),
        Run:   runDeleteRecord,
    })

    searchCmd := &cobra.Command{
        Use:   "search [query]",
        Short: "Search records by context",
        Args:  cobra.ExactArgs(1),
        Run:   runSearchRecords,
    }
    recordsCmd.AddCommand(searchCmd)

    rootCmd.AddCommand(recordsCmd)

    // Import commands
    importCmd := &cobra.Command{
        Use:   "import",
        Short: "Import rules from various sources",
    }

    // Import from file
    importFileCmd := &cobra.Command{
        Use:   "file [filename]",
        Short: "Import rules from ipset save file",
        Args:  cobra.ExactArgs(1),
        Run:   runImportFile,
    }
    importFileCmd.Flags().StringP("context-prefix", "p", "imported", "Context prefix for imported rules")
    importFileCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    importCmd.AddCommand(importFileCmd)

    // Import from /etc/ipset
    importEtcCmd := &cobra.Command{
        Use:   "etc",
        Short: "Import rules from /etc/ipset (default location)",
        Run:   runImportEtc,
    }
    importEtcCmd.Flags().StringP("context-prefix", "p", "etc", "Context prefix for imported rules")
    importEtcCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    importCmd.AddCommand(importEtcCmd)

    // Import from stdin
    importStdinCmd := &cobra.Command{
        Use:   "stdin",
        Short: "Import rules from stdin",
        Run:   runImportStdin,
    }
    importStdinCmd.Flags().StringP("context-prefix", "p", "stdin", "Context prefix for imported rules")
    importStdinCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    importCmd.AddCommand(importStdinCmd)

    // Import from running system
    importSystemCmd := &cobra.Command{
        Use:   "system [setname]",
        Short: "Import rules from running ipset (requires ipset command)",
        Args:  cobra.MaximumNArgs(1),
        Run:   runImportSystem,
    }
    importSystemCmd.Flags().StringP("context-prefix", "p", "system", "Context prefix for imported rules")
    importSystemCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    importCmd.AddCommand(importSystemCmd)

    // Import all (try multiple sources)
    importAllCmd := &cobra.Command{
        Use:   "all",
        Short: "Try to import from all available sources",
        Run:   runImportAll,
    }
    importAllCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    importCmd.AddCommand(importAllCmd)

    rootCmd.AddCommand(importCmd)

    // Config command
    configCmd := &cobra.Command{
        Use:   "config",
        Short: "Manage configuration",
    }
    
    configCmd.AddCommand(&cobra.Command{
        Use:   "set [key] [value]",
        Short: "Set configuration value",
        Args:  cobra.ExactArgs(2),
        Run:   runConfigSet,
    })
    
    configCmd.AddCommand(&cobra.Command{
        Use:   "get [key]",
        Short: "Get configuration value",
        Args:  cobra.ExactArgs(1),
        Run:   runConfigGet,
    })
    
    configCmd.AddCommand(&cobra.Command{
        Use:   "view",
        Short: "View all configuration",
        Run:   runConfigView,
    })
    
    rootCmd.AddCommand(configCmd)

    // Export command
    exportCmd := &cobra.Command{
        Use:   "export [id]",
        Short: "Export record as ipset rule",
        Args:  cobra.MaximumNArgs(1),
        Run:   runExport,
    }
    rootCmd.AddCommand(exportCmd)

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func initConfig() {
    home, err := os.UserHomeDir()
    if err != nil {
        home = "."
    }

    viper.AddConfigPath(home)
    viper.AddConfigPath(".")
    viper.SetConfigName(".ipset-cli")
    viper.SetConfigType("yaml")
    
    viper.SetDefault("api_url", "http://localhost:8080")
    viper.SetDefault("output", "table")
    viper.SetDefault("insecure", false)

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            fmt.Printf("Error reading config file: %v\n", err)
        }
    }
}

func runLogin(cmd *cobra.Command, args []string) {
    apiKey := args[0]
    
    data := map[string]string{"api_key": apiKey}
    jsonData, _ := json.Marshal(data)
    
    resp, err := client.Post(config.APIURL+"/login", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        fmt.Printf("Login failed: %s\n", body)
        return
    }

    var result map[string]string
    json.NewDecoder(resp.Body).Decode(&result)
    
    token := result["token"]
    fmt.Printf("Token: %s\n", token)
    
    // Сохраняем токен в конфиг
    viper.Set("token", token)
    if err := viper.WriteConfig(); err != nil {
        // Если файла нет, создаем новый
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            viper.SafeWriteConfig()
        } else {
            fmt.Printf("Warning: Failed to save token: %v\n", err)
        }
    }
}

func runListRecords(cmd *cobra.Command, args []string) {
    data, err := makeRequest("GET", "/records", nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var records []map[string]interface{}
    json.Unmarshal(data, &records)
    
    outputResults(records)
}

func runGetRecord(cmd *cobra.Command, args []string) {
    id := args[0]
    
    data, err := makeRequest("GET", "/records/"+id, nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var record map[string]interface{}
    json.Unmarshal(data, &record)
    
    outputResults([]map[string]interface{}{record})
}

func runCreateRecord(cmd *cobra.Command, args []string) {
    ip, _ := cmd.Flags().GetString("ip")
    cidr, _ := cmd.Flags().GetString("cidr")
    port, _ := cmd.Flags().GetInt("port")
    protocol, _ := cmd.Flags().GetString("protocol")
    description, _ := cmd.Flags().GetString("description")
    context, _ := cmd.Flags().GetString("context")

    record := map[string]interface{}{
        "ip":          ip,
        "context":     context,
    }
    
    if cidr != "" {
        record["cidr"] = cidr
    }
    if port != 0 {
        record["port"] = port
    }
    if protocol != "" {
        record["protocol"] = protocol
    }
    if description != "" {
        record["description"] = description
    }

    jsonData, _ := json.Marshal(record)
    
    data, err := makeRequestWithBody("POST", "/records", jsonData)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var result map[string]interface{}
    json.Unmarshal(data, &result)
    
    fmt.Println("Record created successfully:")
    outputResults([]map[string]interface{}{result})
}

func runUpdateRecord(cmd *cobra.Command, args []string) {
    id := args[0]
    
    record := make(map[string]interface{})
    
    if ip, _ := cmd.Flags().GetString("ip"); ip != "" {
        record["ip"] = ip
    }
    if cidr, _ := cmd.Flags().GetString("cidr"); cidr != "" {
        record["cidr"] = cidr
    }
    if port, _ := cmd.Flags().GetInt("port"); port != 0 {
        record["port"] = port
    }
    if protocol, _ := cmd.Flags().GetString("protocol"); protocol != "" {
        record["protocol"] = protocol
    }
    if description, _ := cmd.Flags().GetString("description"); description != "" {
        record["description"] = description
    }
    if context, _ := cmd.Flags().GetString("context"); context != "" {
        record["context"] = context
    }

    jsonData, _ := json.Marshal(record)
    
    data, err := makeRequestWithBody("PUT", "/records/"+id, jsonData)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var result map[string]interface{}
    json.Unmarshal(data, &result)
    
    fmt.Println("Record updated successfully:")
    outputResults([]map[string]interface{}{result})
}

func runDeleteRecord(cmd *cobra.Command, args []string) {
    id := args[0]
    
    _, err := makeRequest("DELETE", "/records/"+id, nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Record %s deleted successfully\n", id)
}

func runSearchRecords(cmd *cobra.Command, args []string) {
    query := args[0]
    
    data, err := makeRequest("GET", "/records/search?q="+query, nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var records []map[string]interface{}
    json.Unmarshal(data, &records)
    
    outputResults(records)
}

func runExport(cmd *cobra.Command, args []string) {
    var records []map[string]interface{}
    
    if len(args) == 0 {
        // Export all records
        data, err := makeRequest("GET", "/records", nil)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            return
        }
        json.Unmarshal(data, &records)
    } else {
        // Export single record
        id := args[0]
        data, err := makeRequest("GET", "/records/"+id, nil)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            return
        }
        var record map[string]interface{}
        json.Unmarshal(data, &record)
        records = []map[string]interface{}{record}
    }
    
    outputAsIPSet(records)
}

func runConfigSet(cmd *cobra.Command, args []string) {
    key := args[0]
    value := args[1]
    
    viper.Set(key, value)
    if err := viper.WriteConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            viper.SafeWriteConfig()
        } else {
            fmt.Printf("Error saving config: %v\n", err)
            return
        }
    }
    
    fmt.Printf("Config %s set to %s\n", key, value)
}

func runConfigGet(cmd *cobra.Command, args []string) {
    key := args[0]
    value := viper.Get(key)
    fmt.Printf("%s: %v\n", key, value)
}

func runConfigView(cmd *cobra.Command, args []string) {
    settings := viper.AllSettings()
    yamlData, _ := yaml.Marshal(settings)
    fmt.Println(string(yamlData))
}

func makeRequest(method, path string, body []byte) ([]byte, error) {
    return makeRequestWithBody(method, path, body)
}

func makeRequestWithBody(method, path string, body []byte) ([]byte, error) {
    url := config.APIURL + path
    
    var req *http.Request
    var err error
    
    if body != nil {
        req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
    } else {
        req, err = http.NewRequest(method, url, nil)
    }
    
    if err != nil {
        return nil, err
    }
    
    if config.Token != "" {
        req.Header.Set("Authorization", "Bearer "+config.Token)
    }
    
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
    }
    
    return data, nil
}

func outputResults(records []map[string]interface{}) {
    switch config.Output {
    case "json":
        outputAsJSON(records)
    case "yaml":
        outputAsYAML(records)
    case "ipset":
        outputAsIPSet(records)
    case "table":
        fallthrough
    default:
        outputAsTable(records)
    }
}

func outputAsJSON(data interface{}) {
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        fmt.Printf("Error marshaling JSON: %v\n", err)
        return
    }
    fmt.Println(string(jsonData))
}

func outputAsYAML(data interface{}) {
    yamlData, err := yaml.Marshal(data)
    if err != nil {
        fmt.Printf("Error marshaling YAML: %v\n", err)
        return
    }
    fmt.Println(string(yamlData))
}

func outputAsTable(records []map[string]interface{}) {
    if len(records) == 0 {
        fmt.Println("No records found")
        return
    }
    
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"ID", "IP", "CIDR", "Port", "Protocol", "Description", "Context", "Created"})
    table.SetBorder(false)
    table.SetRowLine(true)
    table.SetColumnSeparator("│")
    table.SetHeaderColor(
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
    )
    
    for _, record := range records {
        created := ""
        if t, ok := record["created_at"].(string); ok {
            if len(t) > 10 {
                created = t[:10]
            } else {
                created = t
            }
        }
        
        table.Append([]string{
            fmt.Sprintf("%v", record["id"]),
            fmt.Sprintf("%v", record["ip"]),
            fmt.Sprintf("%v", record["cidr"]),
            fmt.Sprintf("%v", record["port"]),
            fmt.Sprintf("%v", record["protocol"]),
            fmt.Sprintf("%v", record["description"]),
            truncateString(fmt.Sprintf("%v", record["context"]), 30),
            created,
        })
    }
    
    table.Render()
}

func outputAsIPSet(records []map[string]interface{}) {
    if len(records) == 0 {
        fmt.Println("No records to export")
        return
    }
    
    fmt.Println("#!/bin/bash")
    fmt.Println("# IPSet rules generated by ipset-cli")
    fmt.Println()
    
    // Группируем по протоколам для создания разных sets
    tcpRules := []string{}
    udpRules := []string{}
    otherRules := []string{}
    
    for _, record := range records {
        ip := fmt.Sprintf("%v", record["ip"])
        cidr := fmt.Sprintf("%v", record["cidr"])
        port := fmt.Sprintf("%v", record["port"])
        protocol := fmt.Sprintf("%v", record["protocol"])
        desc := fmt.Sprintf("%v", record["description"])
        id := fmt.Sprintf("%v", record["id"])
        
        // Формируем IP/CIDR
        ipWithCIDR := ip
        if cidr != "" && cidr != "<nil>" && cidr != "0" {
            ipWithCIDR = ip + "/" + cidr
        }
        
        // Создаем комментарий
        comment := fmt.Sprintf("# Rule ID: %s", id)
        if desc != "" && desc != "<nil>" {
            comment += fmt.Sprintf(" - %s", desc)
        }
        
        // Формируем правило в зависимости от протокола и порта
        var rule string
        if protocol != "" && protocol != "<nil>" && port != "" && port != "<nil>" && port != "0" {
            // Правило с протоколом и портом
            rule = fmt.Sprintf("ipset add %s_set %s,%s:%s",
                protocol, ipWithCIDR, protocol, port)
        } else {
            // Простое правило без порта
            rule = fmt.Sprintf("ipset add ip_set %s", ipWithCIDR)
        }
        
        // Добавляем в соответствующую группу
        switch protocol {
        case "tcp":
            tcpRules = append(tcpRules, fmt.Sprintf("%s\n%s", comment, rule))
        case "udp":
            udpRules = append(udpRules, fmt.Sprintf("%s\n%s", comment, rule))
        default:
            otherRules = append(otherRules, fmt.Sprintf("%s\n%s", comment, rule))
        }
    }
    
    // Создаем ipset sets если нужно
    if len(tcpRules) > 0 {
        fmt.Println("# Create TCP set if not exists")
        fmt.Println("ipset create tcp_set hash:ip,port -exist")
        fmt.Println()
        for _, rule := range tcpRules {
            fmt.Println(rule)
        }
        fmt.Println()
    }
    
    if len(udpRules) > 0 {
        fmt.Println("# Create UDP set if not exists")
        fmt.Println("ipset create udp_set hash:ip,port -exist")
        fmt.Println()
        for _, rule := range udpRules {
            fmt.Println(rule)
        }
        fmt.Println()
    }
    
    if len(otherRules) > 0 {
        fmt.Println("# Create generic IP set if not exists")
        fmt.Println("ipset create ip_set hash:ip -exist")
        fmt.Println()
        for _, rule := range otherRules {
            fmt.Println(rule)
        }
        fmt.Println()
    }
    
    fmt.Println("# Example iptables rules:")
    fmt.Println("# iptables -A INPUT -m set --match-set tcp_set src -j ACCEPT")
    fmt.Println("# iptables -A INPUT -m set --match-set udp_set src -j ACCEPT")
    fmt.Println("# iptables -A INPUT -m set --match-set ip_set src -j ACCEPT")
}

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}

func runImportFile(cmd *cobra.Command, args []string) {
    filename := args[0]
    contextPrefix, _ := cmd.Flags().GetString("context-prefix")
    dryRun, _ := cmd.Flags().GetBool("dry-run")
    
    rules, err := parseIPSetFile(filename)
    if err != nil {
        fmt.Printf("Error parsing file: %v\n", err)
        return
    }
    
    importRules(rules, contextPrefix, dryRun)
}

func runImportEtc(cmd *cobra.Command, args []string) {
    etcFile := "/etc/ipset"
    contextPrefix, _ := cmd.Flags().GetString("context-prefix")
    dryRun, _ := cmd.Flags().GetBool("dry-run")
    
    // Проверяем существование файла
    if _, err := os.Stat(etcFile); os.IsNotExist(err) {
        fmt.Printf("File %s does not exist\n", etcFile)
        return
    }
    
    rules, err := parseIPSetFile(etcFile)
    if err != nil {
        fmt.Printf("Error parsing %s: %v\n", etcFile, err)
        return
    }
    
    fmt.Printf("Importing from %s\n", etcFile)
    importRules(rules, contextPrefix, dryRun)
}

func runImportStdin(cmd *cobra.Command, args []string) {
    contextPrefix, _ := cmd.Flags().GetString("context-prefix")
    dryRun, _ := cmd.Flags().GetBool("dry-run")
    
    stat, _ := os.Stdin.Stat()
    if (stat.Mode() & os.ModeCharDevice) != 0 {
        fmt.Println("No data on stdin. Pipe data to stdin or use 'ipset-cli import file'")
        return
    }
    
    data, err := io.ReadAll(os.Stdin)
    if err != nil {
        fmt.Printf("Error reading stdin: %v\n", err)
        return
    }
    
    rules, err := parseIPSetData(string(data), "stdin")
    if err != nil {
        fmt.Printf("Error parsing stdin data: %v\n", err)
        return
    }
    
    fmt.Println("Importing from stdin")
    importRules(rules, contextPrefix, dryRun)
}

func runImportSystem(cmd *cobra.Command, args []string) {
    contextPrefix, _ := cmd.Flags().GetString("context-prefix")
    dryRun, _ := cmd.Flags().GetBool("dry-run")
    
    // Проверяем наличие ipset команды
    if !commandExists("ipset") {
        fmt.Println("ipset command not found in PATH")
        return
    }
    
    var rules []ImportedRule
    
    if len(args) == 0 {
        // Получаем список всех set
        sets, err := getIPSetSets()
        if err != nil {
            fmt.Printf("Error getting ipset sets: %v\n", err)
            return
        }
        
        for _, set := range sets {
            setRules, err := getIPSetRules(set)
            if err != nil {
                fmt.Printf("Warning: failed to get rules for set %s: %v\n", set, err)
                continue
            }
            rules = append(rules, setRules...)
        }
    } else {
        // Получаем правила для конкретного set
        set := args[0]
        setRules, err := getIPSetRules(set)
        if err != nil {
            fmt.Printf("Error getting rules for set %s: %v\n", set, err)
            return
        }
        rules = setRules
    }
    
    fmt.Printf("Importing from running ipset system\n")
    importRules(rules, contextPrefix, dryRun)
}

func runImportAll(cmd *cobra.Command, args []string) {
    dryRun, _ := cmd.Flags().GetBool("dry-run")
    var allRules []ImportedRule
    sources := []string{}
    
    // Пробуем импортировать из /etc/ipset
    if _, err := os.Stat("/etc/ipset"); err == nil {
        rules, err := parseIPSetFile("/etc/ipset")
        if err == nil && len(rules) > 0 {
            allRules = append(allRules, rules...)
            sources = append(sources, "/etc/ipset")
        }
    }
    
    // Пробуем импортировать из текущей системы
    if commandExists("ipset") {
        sets, err := getIPSetSets()
        if err == nil {
            for _, set := range sets {
                rules, err := getIPSetRules(set)
                if err == nil && len(rules) > 0 {
                    allRules = append(allRules, rules...)
                }
            }
            if len(sets) > 0 {
                sources = append(sources, "running system")
            }
        }
    }
    
    // Пробуем найти другие ipset файлы
    commonLocations := []string{
        "/etc/ipset.conf",
        "/etc/ipset/rules",
        "/var/lib/ipset/rules",
        "/etc/network/ipset",
    }
    
    for _, loc := range commonLocations {
        if _, err := os.Stat(loc); err == nil {
            rules, err := parseIPSetFile(loc)
            if err == nil && len(rules) > 0 {
                allRules = append(allRules, rules...)
                sources = append(sources, loc)
            }
        }
    }
    
    if len(allRules) == 0 {
        fmt.Println("No rules found in any source")
        return
    }
    
    fmt.Printf("Found %d rules from sources: %s\n", len(allRules), strings.Join(sources, ", "))
    importRules(allRules, "imported", dryRun)
}

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
    
    // Регулярные выражения для различных форматов ipset
    patterns := []struct {
        name    string
        regex   *regexp.Regexp
        handler func([]string) *ImportedRule
    }{
        {
            name:  "save format",
            regex: regexp.MustCompile(`^add\s+(\S+)\s+(\S+)(?:\s+.*)?$`),
            handler: func(matches []string) *ImportedRule {
                if len(matches) < 3 {
                    return nil
                }
                return parseIPSetEntry(matches[1], matches[2])
            },
        },
        {
            name:  "ipset command",
            regex: regexp.MustCompile(`^ipset\s+add\s+(\S+)\s+(\S+)(?:\s+.*)?$`),
            handler: func(matches []string) *ImportedRule {
                if len(matches) < 3 {
                    return nil
                }
                return parseIPSetEntry(matches[1], matches[2])
            },
        },
        {
            name:  "create command",
            regex: regexp.MustCompile(`^ipset\s+create\s+(\S+)\s+(\S+)(?:\s+.*)?$`),
            handler: func(matches []string) *ImportedRule {
                // Это команда создания set, не импортируем как правило
                return nil
            },
        },
    }
    
    for i, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        for _, pattern := range patterns {
            if matches := pattern.regex.FindStringSubmatch(line); matches != nil {
                rule := pattern.handler(matches)
                if rule != nil {
                    rule.LineNumber = i + 1
                    rule.Description = fmt.Sprintf("Imported from %s line %d", source, i+1)
                    rules = append(rules, *rule)
                }
                break
            }
        }
    }
    
    return rules, nil
}

func parseIPSetEntry(setName, entry string) *ImportedRule {
    rule := &ImportedRule{
        SetName: setName,
    }
    
    // Парсим entry в формате: IP,IP/CIDR,IP:port,IP/CIDR,protocol:port
    parts := strings.Split(entry, ",")
    if len(parts) == 0 {
        return nil
    }
    
    ipPart := parts[0]
    
    // Проверяем на наличие порта через двоеточие
    if strings.Contains(ipPart, ":") {
        ipPort := strings.Split(ipPart, ":")
        if len(ipPort) == 2 {
            rule.IP = ipPort[0]
            if port, err := strconv.Atoi(ipPort[1]); err == nil {
                rule.Port = port
            }
        }
    } else {
        rule.IP = ipPart
    }
    
    // Проверяем CIDR
    if strings.Contains(rule.IP, "/") {
        ipCIDR := strings.Split(rule.IP, "/")
        rule.IP = ipCIDR[0]
        rule.CIDR = ipCIDR[1]
    }
    
    // Если есть дополнительные части, возможно там протокол:порт
    if len(parts) > 1 {
        protoPort := parts[1]
        if strings.Contains(protoPort, ":") {
            pp := strings.Split(protoPort, ":")
            if len(pp) == 2 {
                rule.Protocol = pp[0]
                if port, err := strconv.Atoi(pp[1]); err == nil && rule.Port == 0 {
                    rule.Port = port
                }
            }
        }
    }
    
    // Определяем протокол из имени set если возможно
    if rule.Protocol == "" {
        if strings.Contains(setName, "tcp") {
            rule.Protocol = "tcp"
        } else if strings.Contains(setName, "udp") {
            rule.Protocol = "udp"
        }
    }
    
    rule.Context = fmt.Sprintf("%s:%s", rule.SetName, rule.IP)
    if rule.Port != 0 {
        rule.Context += fmt.Sprintf(":%d", rule.Port)
    }
    if rule.Protocol != "" {
        rule.Context += fmt.Sprintf(":%s", rule.Protocol)
    }
    
    return rule
}

func importRules(rules []ImportedRule, contextPrefix string, dryRun bool) {
    if len(rules) == 0 {
        fmt.Println("No rules to import")
        return
    }
    
    fmt.Printf("Found %d rules to import\n", len(rules))
    
    if dryRun {
        fmt.Println("\nDRY RUN - Rules that would be imported:")
        for i, rule := range rules {
            fmt.Printf("\n--- Rule %d ---\n", i+1)
            fmt.Printf("  IP: %s\n", rule.IP)
            if rule.CIDR != "" {
                fmt.Printf("  CIDR: %s\n", rule.CIDR)
            }
            if rule.Port != 0 {
                fmt.Printf("  Port: %d\n", rule.Port)
            }
            if rule.Protocol != "" {
                fmt.Printf("  Protocol: %s\n", rule.Protocol)
            }
            fmt.Printf("  Set: %s\n", rule.SetName)
            fmt.Printf("  Context: %s\n", rule.Context)
            fmt.Printf("  Description: %s\n", rule.Description)
        }
        return
    }
    
    // Проверяем авторизацию
    if config.Token == "" {
        fmt.Println("Error: Not authenticated. Please login first using 'ipset-cli login'")
        return
    }
    
    var successCount, failCount int
    
    for _, rule := range rules {
        record := map[string]interface{}{
            "ip":          rule.IP,
            "context":     fmt.Sprintf("%s:%s", contextPrefix, rule.Context),
            "description": rule.Description,
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
            fmt.Printf("❌ Failed to import rule %s: %v\n", rule.IP, err)
            failCount++
        } else {
            fmt.Printf("✅ Imported %s", rule.IP)
            if rule.Port != 0 {
                fmt.Printf(":%d", rule.Port)
            }
            fmt.Println()
            successCount++
        }
    }
    
    fmt.Printf("\nImport completed: %d successful, %d failed\n", successCount, failCount)
}

func getIPSetSets() ([]string, error) {
    cmd := execCommand("ipset", "list", "-n")
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
    cmd := execCommand("ipset", "save", setName)
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    return parseIPSetData(string(output), fmt.Sprintf("system:%s", setName))
}

func commandExists(cmd string) bool {
    _, err := execCommand("which", cmd).Output()
    return err == nil
}
