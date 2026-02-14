// cmd/cli/import.go
package main

import (
    //"encoding/json"
    "fmt"
    "io"
    "os"
    //"os/exec"
    //"regexp"
    //"strconv"
    "strings"
    
    
    "github.com/spf13/cobra"
)

type ImportedRule struct {
    SetName     string
    SetType     string
    SetOptions  string
    IP          string
    CIDR        string
    Port        int
    Protocol    string
    Description string
    Context     string
    LineNumber  int
}

func NewImportCmd() *cobra.Command {
    cmd := &cobra.Command{
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
    cmd.AddCommand(importFileCmd)

    // Import from /etc/ipset
    importEtcCmd := &cobra.Command{
        Use:   "etc",
        Short: "Import rules from /etc/ipset (default location)",
        Run:   runImportEtc,
    }
    importEtcCmd.Flags().StringP("context-prefix", "p", "etc", "Context prefix for imported rules")
    importEtcCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    cmd.AddCommand(importEtcCmd)

    // Import from stdin
    importStdinCmd := &cobra.Command{
        Use:   "stdin",
        Short: "Import rules from stdin",
        Run:   runImportStdin,
    }
    importStdinCmd.Flags().StringP("context-prefix", "p", "stdin", "Context prefix for imported rules")
    importStdinCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    cmd.AddCommand(importStdinCmd)

    // Import from running system
    importSystemCmd := &cobra.Command{
        Use:   "system [setname]",
        Short: "Import rules from running ipset (requires ipset command)",
        Args:  cobra.MaximumNArgs(1),
        Run:   runImportSystem,
    }
    importSystemCmd.Flags().StringP("context-prefix", "p", "system", "Context prefix for imported rules")
    importSystemCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    cmd.AddCommand(importSystemCmd)

    // Import all
    importAllCmd := &cobra.Command{
        Use:   "all",
        Short: "Try to import from all available sources",
        Run:   runImportAll,
    }
    importAllCmd.Flags().BoolP("dry-run", "d", false, "Dry run - show what would be imported")
    cmd.AddCommand(importAllCmd)

    return cmd
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
    
    if !commandExists("ipset") {
        fmt.Println("ipset command not found in PATH")
        return
    }
    
    var rules []ImportedRule
    
    if len(args) == 0 {
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