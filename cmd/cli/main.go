// cmd/cli/main.go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/olekukonko/tablewriter"
    "github.com/urfave/cli/v2"
    "github.com/fatih/color"
)

type Config struct {
    APIURL   string `json:"api_url"`
    Token    string `json:"token"`
    Insecure bool   `json:"insecure"`
}

var config Config
var configFile = os.Getenv("HOME") + "/.ipset-cli-config.json"

func main() {
    // Загружаем конфигурацию
    loadConfig()

    app := &cli.App{
        Name:     "ipset-cli",
        Usage:    "CLI for managing IPSet API",
        Version:  "1.0.0",
        Commands: []*cli.Command{
            // Команды для авторизации
            authCommand(),
            
            // Команды для управления записями
            recordsCommand(),
            
            // Команды для управления ipset
            ipsetsCommand(),
            
            // Команды для управления iptables правилами
            iptablesCommand(),
            
            // Команды для конфигурации
            configCommand(),
            
            // Команды для импорта/экспорта
            importCommand(),
            exportCommand(),
            
            // Команды для применения правил
            applyCommand(),
            
            // Команды для мониторинга
            statsCommand(),
        },
    }

    if err := app.Run(os.Args); err != nil {
        color.Red("Error: %v", err)
        os.Exit(1)
    }
}

// Загрузка конфигурации
func loadConfig() {
    data, err := os.ReadFile(configFile)
    if err != nil {
        config = Config{
            APIURL: "http://localhost:8080",
        }
        return
    }
    json.Unmarshal(data, &config)
}

// Сохранение конфигурации
func saveConfig() error {
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(configFile, data, 0644)
}

// Команды для авторизации
func authCommand() *cli.Command {
    return &cli.Command{
        Name:  "auth",
        Usage: "Authentication commands",
        Subcommands: []*cli.Command{
            {
                Name:  "login",
                Usage: "Login with API key",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "key",
                        Aliases:  []string{"k"},
                        Usage:    "API key",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    key := c.String("key")
                    
                    data := map[string]string{"api_key": key}
                    jsonData, _ := json.Marshal(data)
                    
                    resp, err := http.Post(config.APIURL+"/login", "application/json", bytes.NewBuffer(jsonData))
                    if err != nil {
                        return err
                    }
                    defer resp.Body.Close()
                    
                    var result map[string]string
                    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
                        return err
                    }
                    
                    if token, ok := result["token"]; ok {
                        config.Token = token
                        saveConfig()
                        color.Green("✓ Successfully logged in!")
                        return nil
                    }
                    
                    return fmt.Errorf("login failed: %s", result["error"])
                },
            },
            {
                Name:  "logout",
                Usage: "Logout and clear token",
                Action: func(c *cli.Context) error {
                    config.Token = ""
                    saveConfig()
                    color.Green("✓ Logged out successfully")
                    return nil
                },
            },
            {
                Name:  "status",
                Usage: "Check authentication status",
                Action: func(c *cli.Context) error {
                    if config.Token != "" {
                        color.Green("✓ Authenticated")
                        fmt.Printf("API URL: %s\n", config.APIURL)
                    } else {
                        color.Yellow("✗ Not authenticated")
                    }
                    return nil
                },
            },
        },
    }
}

// Команды для управления записями
func recordsCommand() *cli.Command {
    return &cli.Command{
        Name:  "records",
        Usage: "Manage IPSet records",
        Subcommands: []*cli.Command{
            {
                Name:  "list",
                Usage: "List all records",
                Action: func(c *cli.Context) error {
                    records, err := makeRequest("GET", "/records", nil)
                    if err != nil {
                        return err
                    }
                    
                    table := tablewriter.NewWriter(os.Stdout)
                    table.SetHeader([]string{"ID", "IP", "CIDR", "Port", "Protocol", "Description", "Context", "Created"})
                    
                    for _, record := range records.([]interface{}) {
                        r := record.(map[string]interface{})
                        created, _ := time.Parse(time.RFC3339, r["created_at"].(string))
                        table.Append([]string{
                            fmt.Sprintf("%.0f", r["id"].(float64)),
                            r["ip"].(string),
                            getString(r["cidr"]),
                            getString(r["port"]),
                            getString(r["protocol"]),
                            truncateString(r["description"].(string), 20),
                            truncateString(r["context"].(string), 20),
                            created.Format("2006-01-02 15:04"),
                        })
                    }
                    
                    table.Render()
                    return nil
                },
            },
            {
                Name:  "get",
                Usage: "Get record by ID",
                Flags: []cli.Flag{
                    &cli.IntFlag{
                        Name:     "id",
                        Usage:    "Record ID (6-digit number)",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    id := c.Int("id")
                    record, err := makeRequest("GET", fmt.Sprintf("/records/%d", id), nil)
                    if err != nil {
                        return err
                    }
                    
                    printJSON(record)
                    return nil
                },
            },
            {
                Name:  "create",
                Usage: "Create new record",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "ip",
                        Usage:    "IP address",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:  "cidr",
                        Usage: "CIDR notation",
                    },
                    &cli.IntFlag{
                        Name:  "port",
                        Usage: "Port number",
                    },
                    &cli.StringFlag{
                        Name:  "protocol",
                        Usage: "Protocol (tcp/udp)",
                    },
                    &cli.StringFlag{
                        Name:     "description",
                        Usage:    "Description",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:     "context",
                        Usage:    "Context",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    data := map[string]interface{}{
                        "ip":          c.String("ip"),
                        "cidr":        c.String("cidr"),
                        "port":        c.Int("port"),
                        "protocol":    c.String("protocol"),
                        "description": c.String("description"),
                        "context":     c.String("context"),
                    }
                    
                    record, err := makeRequest("POST", "/records", data)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ Record created successfully")
                    printJSON(record)
                    return nil
                },
            },
            {
                Name:  "update",
                Usage: "Update record",
                Flags: []cli.Flag{
                    &cli.IntFlag{
                        Name:     "id",
                        Usage:    "Record ID",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:  "ip",
                        Usage: "IP address",
                    },
                    &cli.StringFlag{
                        Name:  "cidr",
                        Usage: "CIDR notation",
                    },
                    &cli.IntFlag{
                        Name:  "port",
                        Usage: "Port number",
                    },
                    &cli.StringFlag{
                        Name:  "protocol",
                        Usage: "Protocol",
                    },
                    &cli.StringFlag{
                        Name:  "description",
                        Usage: "Description",
                    },
                    &cli.StringFlag{
                        Name:  "context",
                        Usage: "Context",
                    },
                },
                Action: func(c *cli.Context) error {
                    id := c.Int("id")
                    data := make(map[string]interface{})
                    
                    if c.IsSet("ip") {
                        data["ip"] = c.String("ip")
                    }
                    if c.IsSet("cidr") {
                        data["cidr"] = c.String("cidr")
                    }
                    if c.IsSet("port") {
                        data["port"] = c.Int("port")
                    }
                    if c.IsSet("protocol") {
                        data["protocol"] = c.String("protocol")
                    }
                    if c.IsSet("description") {
                        data["description"] = c.String("description")
                    }
                    if c.IsSet("context") {
                        data["context"] = c.String("context")
                    }
                    
                    record, err := makeRequest("PUT", fmt.Sprintf("/records/%d", id), data)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ Record updated successfully")
                    printJSON(record)
                    return nil
                },
            },
            {
                Name:  "delete",
                Usage: "Delete record",
                Flags: []cli.Flag{
                    &cli.IntFlag{
                        Name:     "id",
                        Usage:    "Record ID",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    id := c.Int("id")
                    
                    _, err := makeRequest("DELETE", fmt.Sprintf("/records/%d", id), nil)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ Record deleted successfully")
                    return nil
                },
            },
            {
                Name:  "search",
                Usage: "Search records",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "query",
                        Aliases:  []string{"q"},
                        Usage:    "Search query",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    query := c.String("query")
                    
                    records, err := makeRequest("GET", fmt.Sprintf("/records/search?q=%s", query), nil)
                    if err != nil {
                        return err
                    }
                    
                    table := tablewriter.NewWriter(os.Stdout)
                    table.SetHeader([]string{"ID", "IP", "Description", "Context"})
                    
                    for _, record := range records.([]interface{}) {
                        r := record.(map[string]interface{})
                        table.Append([]string{
                            fmt.Sprintf("%.0f", r["id"].(float64)),
                            r["ip"].(string),
                            truncateString(r["description"].(string), 30),
                            truncateString(r["context"].(string), 30),
                        })
                    }
                    
                    table.Render()
                    return nil
                },
            },
        },
    }
}

// Команды для управления ipset
func ipsetsCommand() *cli.Command {
    return &cli.Command{
        Name:  "ipsets",
        Usage: "Manage IPSets",
        Subcommands: []*cli.Command{
            {
                Name:  "list",
                Usage: "List all ipsets",
                Action: func(c *cli.Context) error {
                    sets, err := makeRequest("GET", "/ipsets", nil)
                    if err != nil {
                        return err
                    }
                    
                    table := tablewriter.NewWriter(os.Stdout)
                    table.SetHeader([]string{"Name", "Type", "Family", "HashSize", "MaxElem", "Entries", "Description"})
                    
                    for _, set := range sets.([]interface{}) {
                        s := set.(map[string]interface{})
                        entries := s["entries"].([]interface{})
                        table.Append([]string{
                            s["name"].(string),
                            string(s["type"].(string)),
                            string(s["family"].(string)),
                            fmt.Sprintf("%.0f", s["hashsize"].(float64)),
                            fmt.Sprintf("%.0f", s["maxelem"].(float64)),
                            fmt.Sprintf("%d", len(entries)),
                            truncateString(s["description"].(string), 20),
                        })
                    }
                    
                    table.Render()
                    return nil
                },
            },
            {
                Name:  "create",
                Usage: "Create new ipset",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "name",
                        Usage:    "Set name",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:     "type",
                        Usage:    "Set type (hash:net, hash:ip, hash:ip,port)",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:  "family",
                        Usage: "Address family (inet/inet6)",
                        Value: "inet",
                    },
                    &cli.IntFlag{
                        Name:  "hashsize",
                        Usage: "Hash size",
                        Value: 1024,
                    },
                    &cli.IntFlag{
                        Name:  "maxelem",
                        Usage: "Maximum elements",
                        Value: 65536,
                    },
                    &cli.StringFlag{
                        Name:  "description",
                        Usage: "Description",
                    },
                },
                Action: func(c *cli.Context) error {
                    data := map[string]interface{}{
                        "name":        c.String("name"),
                        "type":        c.String("type"),
                        "family":      c.String("family"),
                        "hashsize":    c.Int("hashsize"),
                        "maxelem":     c.Int("maxelem"),
                        "description": c.String("description"),
                    }
                    
                    set, err := makeRequest("POST", "/ipsets", data)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ IPSet created successfully")
                    printJSON(set)
                    return nil
                },
            },
            {
                Name:  "entries",
                Usage: "Manage ipset entries",
                Subcommands: []*cli.Command{
                    {
                        Name:  "add",
                        Usage: "Add entry to ipset",
                        Flags: []cli.Flag{
                            &cli.StringFlag{
                                Name:     "set",
                                Usage:    "IPSet name",
                                Required: true,
                            },
                            &cli.StringFlag{
                                Name:     "value",
                                Usage:    "Entry value (e.g., 192.168.1.0/24 or 192.168.1.1,tcp:80)",
                                Required: true,
                            },
                            &cli.StringFlag{
                                Name:  "comment",
                                Usage: "Comment",
                            },
                        },
                        Action: func(c *cli.Context) error {
                            data := map[string]interface{}{
                                "value":   c.String("value"),
                                "comment": c.String("comment"),
                            }
                            
                            entry, err := makeRequest("POST", fmt.Sprintf("/ipsets/%s/entries", c.String("set")), data)
                            if err != nil {
                                return err
                            }
                            
                            color.Green("✓ Entry added successfully")
                            printJSON(entry)
                            return nil
                        },
                    },
                    {
                        Name:  "list",
                        Usage: "List entries in ipset",
                        Flags: []cli.Flag{
                            &cli.StringFlag{
                                Name:     "set",
                                Usage:    "IPSet name",
                                Required: true,
                            },
                        },
                        Action: func(c *cli.Context) error {
                            entries, err := makeRequest("GET", fmt.Sprintf("/ipsets/%s/entries", c.String("set")), nil)
                            if err != nil {
                                return err
                            }
                            
                            table := tablewriter.NewWriter(os.Stdout)
                            table.SetHeader([]string{"ID", "Value", "Comment", "Created"})
                            
                            for _, entry := range entries.([]interface{}) {
                                e := entry.(map[string]interface{})
                                created, _ := time.Parse(time.RFC3339, e["created_at"].(string))
                                table.Append([]string{
                                    fmt.Sprintf("%.0f", e["id"].(float64)),
                                    e["value"].(string),
                                    getString(e["comment"]),
                                    created.Format("2006-01-02 15:04"),
                                })
                            }
                            
                            table.Render()
                            return nil
                        },
                    },
                    {
                        Name:  "remove",
                        Usage: "Remove entry from ipset",
                        Flags: []cli.Flag{
                            &cli.IntFlag{
                                Name:     "id",
                                Usage:    "Entry ID",
                                Required: true,
                            },
                        },
                        Action: func(c *cli.Context) error {
                            id := c.Int("id")
                            
                            _, err := makeRequest("DELETE", fmt.Sprintf("/ipsets/entries/%d", id), nil)
                            if err != nil {
                                return err
                            }
                            
                            color.Green("✓ Entry removed successfully")
                            return nil
                        },
                    },
                },
            },
            {
                Name:  "delete",
                Usage: "Delete ipset",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "name",
                        Usage:    "IPSet name",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    name := c.String("name")
                    
                    _, err := makeRequest("DELETE", fmt.Sprintf("/ipsets/%s", name), nil)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ IPSet deleted successfully")
                    return nil
                },
            },
            {
                Name:  "search",
                Usage: "Search ipsets",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "query",
                        Aliases:  []string{"q"},
                        Usage:    "Search query",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    query := c.String("query")
                    
                    sets, err := makeRequest("GET", fmt.Sprintf("/ipsets/search?q=%s", query), nil)
                    if err != nil {
                        return err
                    }
                    
                    printJSON(sets)
                    return nil
                },
            },
        },
    }
}

// Команды для управления iptables
func iptablesCommand() *cli.Command {
    return &cli.Command{
        Name:  "iptables",
        Usage: "Manage iptables rules",
        Subcommands: []*cli.Command{
            {
                Name:  "list",
                Usage: "List all iptables rules",
                Action: func(c *cli.Context) error {
                    rules, err := makeRequest("GET", "/iptables/rules", nil)
                    if err != nil {
                        return err
                    }
                    
                    table := tablewriter.NewWriter(os.Stdout)
                    table.SetHeader([]string{"ID", "Chain", "Interface", "Protocol", "Src Sets", "Dst Sets", "Action", "Description"})
                    
                    for _, rule := range rules.([]interface{}) {
                        r := rule.(map[string]interface{})
                        table.Append([]string{
                            fmt.Sprintf("%.0f", r["id"].(float64)),
                            r["chain"].(string),
                            getString(r["interface"]),
                            getString(r["protocol"]),
                            fmt.Sprintf("%v", r["src_sets"]),
                            fmt.Sprintf("%v", r["dst_sets"]),
                            r["action"].(string),
                            truncateString(r["description"].(string), 20),
                        })
                    }
                    
                    table.Render()
                    return nil
                },
            },
            {
                Name:  "create",
                Usage: "Create iptables rule",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:     "chain",
                        Usage:    "Chain name",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:  "interface",
                        Usage: "Network interface",
                    },
                    &cli.StringFlag{
                        Name:  "protocol",
                        Usage: "Protocol",
                    },
                    &cli.StringSliceFlag{
                        Name:  "src-set",
                        Usage: "Source set names",
                    },
                    &cli.StringSliceFlag{
                        Name:  "dst-set",
                        Usage: "Destination set names",
                    },
                    &cli.StringFlag{
                        Name:     "action",
                        Usage:    "Action (ACCEPT/DROP/REJECT)",
                        Required: true,
                    },
                    &cli.StringFlag{
                        Name:  "description",
                        Usage: "Description",
                    },
                    &cli.IntFlag{
                        Name:  "position",
                        Usage: "Position in chain",
                    },
                },
                Action: func(c *cli.Context) error {
                    data := map[string]interface{}{
                        "chain":       c.String("chain"),
                        "interface":   c.String("interface"),
                        "protocol":    c.String("protocol"),
                        "src_sets":    c.StringSlice("src-set"),
                        "dst_sets":    c.StringSlice("dst-set"),
                        "action":      c.String("action"),
                        "description": c.String("description"),
                        "position":    c.Int("position"),
                    }
                    
                    rule, err := makeRequest("POST", "/iptables/rules", data)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ Rule created successfully")
                    printJSON(rule)
                    return nil
                },
            },
            {
                Name:  "delete",
                Usage: "Delete iptables rule",
                Flags: []cli.Flag{
                    &cli.IntFlag{
                        Name:     "id",
                        Usage:    "Rule ID",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    id := c.Int("id")
                    
                    _, err := makeRequest("DELETE", fmt.Sprintf("/iptables/rules/%d", id), nil)
                    if err != nil {
                        return err
                    }
                    
                    color.Green("✓ Rule deleted successfully")
                    return nil
                },
            },
            {
                Name:  "generate",
                Usage: "Generate iptables command",
                Flags: []cli.Flag{
                    &cli.IntFlag{
                        Name:     "id",
                        Usage:    "Rule ID",
                        Required: true,
                    },
                },
                Action: func(c *cli.Context) error {
                    id := c.Int("id")
                    
                    result, err := makeRequest("GET", fmt.Sprintf("/generate/iptables/%d", id), nil)
                    if err != nil {
                        return err
                    }
                    
                    cmd := result.(map[string]interface{})["command"]
                    color.Cyan("Generated command:")
                    fmt.Println(cmd)
                    return nil
                },
            },
        },
    }
}

// Команды для конфигурации
func configCommand() *cli.Command {
    return &cli.Command{
        Name:  "config",
        Usage: "Configuration commands",
        Subcommands: []*cli.Command{
            {
                Name:  "set",
                Usage: "Set configuration value",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:  "url",
                        Usage: "API URL",
                    },
                    &cli.BoolFlag{
                        Name:  "insecure",
                        Usage: "Skip TLS verification",
                    },
                },
                Action: func(c *cli.Context) error {
                    if c.IsSet("url") {
                        config.APIURL = c.String("url")
                    }
                    if c.IsSet("insecure") {
                        config.Insecure = c.Bool("insecure")
                    }
                    
                    if err := saveConfig(); err != nil {
                        return err
                    }
                    
                    color.Green("✓ Configuration saved")
                    return nil
                },
            },
            {
                Name:  "show",
                Usage: "Show current configuration",
                Action: func(c *cli.Context) error {
                    table := tablewriter.NewWriter(os.Stdout)
                    table.SetHeader([]string{"Setting", "Value"})
                    table.Append([]string{"API URL", config.APIURL})
                    table.Append([]string{"Token", maskString(config.Token, 10)})
                    table.Append([]string{"Insecure", fmt.Sprintf("%v", config.Insecure)})
                    table.Render()
                    return nil
                },
            },
        },
    }
}

// Команды для импорта/экспорта
func importCommand() *cli.Command {
    return &cli.Command{
        Name:  "import",
        Usage: "Import configuration from file",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:     "file",
                Aliases:  []string{"f"},
                Usage:    "File to import",
                Required: true,
            },
        },
        Action: func(c *cli.Context) error {
            filename := c.String("file")
            
            data, err := os.ReadFile(filename)
            if err != nil {
                return err
            }
            
            importData := map[string]interface{}{
                "config": string(data),
            }
            
            result, err := makeRequest("POST", "/import", importData)
            if err != nil {
                return err
            }
            
            color.Green("✓ Configuration imported successfully")
            printJSON(result)
            return nil
        },
    }
}

func exportCommand() *cli.Command {
    return &cli.Command{
        Name:  "export",
        Usage: "Export configuration to file",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:     "file",
                Aliases:  []string{"f"},
                Usage:    "Output file",
                Required: true,
            },
            &cli.BoolFlag{
                Name:  "include-iptables",
                Usage: "Include iptables rules",
                Value: true,
            },
        },
        Action: func(c *cli.Context) error {
            filename := c.String("file")
            
            // Получаем все ipsets
            sets, err := makeRequest("GET", "/ipsets", nil)
            if err != nil {
                return err
            }
            
            var output strings.Builder
            
            // Экспортируем ipsets
            for _, set := range sets.([]interface{}) {
                s := set.(map[string]interface{})
                output.WriteString(fmt.Sprintf("create %s %s family %s hashsize %.0f maxelem %.0f\n",
                    s["name"], s["type"], s["family"], s["hashsize"], s["maxelem"]))
                
                // Экспортируем entries
                entries, _ := makeRequest("GET", fmt.Sprintf("/ipsets/%s/entries", s["name"]), nil)
                for _, entry := range entries.([]interface{}) {
                    e := entry.(map[string]interface{})
                    output.WriteString(fmt.Sprintf("add %s %s\n", s["name"], e["value"]))
                }
                output.WriteString("\n")
            }
            
            // Экспортируем iptables правила если нужно
            if c.Bool("include-iptables") {
                rules, err := makeRequest("GET", "/iptables/rules", nil)
                if err == nil {
                    for _, rule := range rules.([]interface{}) {
                        r := rule.(map[string]interface{})
                        // Генерируем iptables команду
                        cmd, _ := makeRequest("GET", fmt.Sprintf("/generate/iptables/%.0f", r["id"]), nil)
                        output.WriteString(cmd.(map[string]interface{})["command"].(string) + "\n")
                    }
                }
            }
            
            if err := os.WriteFile(filename, []byte(output.String()), 0644); err != nil {
                return err
            }
            
            color.Green("✓ Configuration exported to %s", filename)
            return nil
        },
    }
}

// Команда для применения правил
func applyCommand() *cli.Command {
    return &cli.Command{
        Name:  "apply",
        Usage: "Apply configuration",
        Flags: []cli.Flag{
            &cli.BoolFlag{
                Name:  "dry-run",
                Usage: "Perform dry run without actual changes",
            },
            &cli.StringFlag{
                Name:  "file",
                Aliases: []string{"f"},
                Usage: "Configuration file to apply",
            },
        },
        Action: func(c *cli.Context) error {
            var ipsetCommands []interface{}
            var iptablesCommands []string
            
            if c.IsSet("file") {
                // Загружаем конфигурацию из файла
                data, err := os.ReadFile(c.String("file"))
                if err != nil {
                    return err
                }
                
                // Парсим файл и создаем команды
                lines := strings.Split(string(data), "\n")
                for _, line := range lines {
                    line = strings.TrimSpace(line)
                    if line == "" || strings.HasPrefix(line, "#") {
                        continue
                    }
                    
                    if strings.HasPrefix(line, "iptables ") {
                        iptablesCommands = append(iptablesCommands, line)
                    } else if strings.HasPrefix(line, "create ") || strings.HasPrefix(line, "add ") {
                        parts := strings.Fields(line)
                        if len(parts) >= 2 {
                            ipsetCommands = append(ipsetCommands, map[string]interface{}{
                                "command": parts[0],
                                "set_name": parts[1],
                                "args": parts[2:],
                            })
                        }
                    }
                }
            }
            
            data := map[string]interface{}{
                "ipset_commands":    ipsetCommands,
                "iptables_commands": iptablesCommands,
            }
            
            url := "/apply"
            if c.Bool("dry-run") {
                url += "?dry_run=true"
            }
            
            result, err := makeRequest("POST", url, data)
            if err != nil {
                return err
            }
            
            printJSON(result)
            return nil
        },
    }
}

// Команда для статистики
func statsCommand() *cli.Command {
    return &cli.Command{
        Name:  "stats",
        Usage: "Show statistics",
        Action: func(c *cli.Context) error {
            // Получаем статистику по записям
            records, _ := makeRequest("GET", "/records", nil)
            sets, _ := makeRequest("GET", "/ipsets", nil)
            rules, _ := makeRequest("GET", "/iptables/rules", nil)
            
            table := tablewriter.NewWriter(os.Stdout)
            table.SetHeader([]string{"Metric", "Count"})
            table.Append([]string{"Total Records", fmt.Sprintf("%d", len(records.([]interface{})))})
            table.Append([]string{"Total IPSets", fmt.Sprintf("%d", len(sets.([]interface{})))})
            table.Append([]string{"Total IPTables Rules", fmt.Sprintf("%d", len(rules.([]interface{})))})
            
            // Подсчет общего количества entries
            totalEntries := 0
            for _, set := range sets.([]interface{}) {
                s := set.(map[string]interface{})
                entries := s["entries"].([]interface{})
                totalEntries += len(entries)
            }
            table.Append([]string{"Total IPSet Entries", fmt.Sprintf("%d", totalEntries)})
            
            table.Render()
            return nil
        },
    }
}

// Вспомогательные функции
func makeRequest(method, path string, body interface{}) (interface{}, error) {
    var jsonData []byte
    var err error
    
    if body != nil {
        jsonData, err = json.Marshal(body)
        if err != nil {
            return nil, err
        }
    }
    
    req, err := http.NewRequest(method, config.APIURL+path, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    if config.Token != "" {
        req.Header.Set("Authorization", "Bearer "+config.Token)
    }
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode >= 400 {
        var errResp map[string]string
        if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
            if msg, ok := errResp["error"]; ok {
                return nil, fmt.Errorf(msg)
            }
        }
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
    }
    
    var result interface{}
    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return nil, err
    }
    
    return result, nil
}

func printJSON(v interface{}) {
    data, _ := json.MarshalIndent(v, "", "  ")
    fmt.Println(string(data))
}

func getString(v interface{}) string {
    if v == nil {
        return ""
    }
    return fmt.Sprintf("%v", v)
}

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}

func maskString(s string, visible int) string {
    if len(s) <= visible {
        return s
    }
    return s[:visible] + "..."
}