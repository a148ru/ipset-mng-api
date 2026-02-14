// cmd/cli/sets.go (исправленная версия со всеми командами)
package main

import (
    "encoding/json"
    "fmt"
    
    "github.com/olekukonko/tablewriter"
    "github.com/spf13/cobra"
)

func NewSetsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "sets",
        Short: "Manage IPSet sets",
        Long:  `List, get, delete and export IPSet sets`,
    }

    // Добавляем все подкоманды для sets
    cmd.AddCommand(NewListSetsCmd())
    cmd.AddCommand(NewGetSetCmd())
    cmd.AddCommand(NewDeleteSetCmd())
    cmd.AddCommand(NewExportSetCmd())  // Это правильное название команды

    return cmd
}

func NewListSetsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all sets",
        Run:   runListSets,
    }
}

func NewGetSetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get [set-name]",
        Short: "Get details of a specific set",
        Args:  cobra.ExactArgs(1),
        Run:   runGetSet,
    }
}

func NewDeleteSetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "delete [set-name]",
        Short: "Delete a set and all its records",
        Args:  cobra.ExactArgs(1),
        Run:   runDeleteSet,
    }
}

func NewExportSetCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "export [set-name]",
        Short: "Export a set as ipset rules",
        Long:  `Export a specific set in various formats (ipset, json, yaml)`,
        Args:  cobra.ExactArgs(1),
        Run:   runExportSet,
    }
    
    cmd.Flags().StringP("format", "f", "ipset", "Export format (ipset, json, yaml)")
    
    return cmd
}

func runListSets(cmd *cobra.Command, args []string) {
    data, err := makeRequest("GET", "/sets", nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var sets []map[string]interface{}
    if err := json.Unmarshal(data, &sets); err != nil {
        fmt.Printf("Error parsing response: %v\n", err)
        return
    }

    switch config.Output {
    case "json":
        outputAsJSON(sets)
    case "yaml":
        outputAsYAML(sets)
    case "table":
        fallthrough
    default:
        table := tablewriter.NewWriter(cmd.OutOrStdout())
        table.SetHeader([]string{"Set Name", "Type", "Records", "Created", "Updated"})
        table.SetBorder(false)
        table.SetRowLine(true)
        table.SetColumnSeparator("│")
        table.SetHeaderColor(
            tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
            tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
            tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
            tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
            tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
        )
        
        for _, set := range sets {
            var recordCount int
            if records, ok := set["records"].([]interface{}); ok {
                recordCount = len(records)
            }
            
            created := formatDate(set["created_at"])
            updated := formatDate(set["updated_at"])
            
            table.Append([]string{
                fmt.Sprintf("%v", set["name"]),
                fmt.Sprintf("%v", set["type"]),
                fmt.Sprintf("%d", recordCount),
                created,
                updated,
            })
        }
        
        table.Render()
    }
}

func runGetSet(cmd *cobra.Command, args []string) {
    setName := args[0]
    
    data, err := makeRequest("GET", "/sets/"+setName, nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    var set map[string]interface{}
    if err := json.Unmarshal(data, &set); err != nil {
        fmt.Printf("Error parsing response: %v\n", err)
        return
    }

    outputResults([]map[string]interface{}{set})
}

func runDeleteSet(cmd *cobra.Command, args []string) {
    setName := args[0]
    
    _, err := makeRequest("DELETE", "/sets/"+setName, nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Set %s deleted successfully\n", setName)
}

func runExportSet(cmd *cobra.Command, args []string) {
    setName := args[0]
    format, _ := cmd.Flags().GetString("format")
    
    data, err := makeRequest("GET", fmt.Sprintf("/sets/%s/export?format=%s", setName, format), nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    if format == "ipset" {
        fmt.Print(string(data))
    } else {
        var result map[string]interface{}
        if err := json.Unmarshal(data, &result); err != nil {
            // Пробуем как массив
            var resultArray []map[string]interface{}
            if err2 := json.Unmarshal(data, &resultArray); err2 == nil {
                outputResults(resultArray)
                return
            }
            fmt.Printf("Error parsing response: %v\n", err)
            return
        }
        outputResults([]map[string]interface{}{result})
    }
}