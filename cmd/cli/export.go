// cmd/cli/export.go (существующий файл для экспорта записей)
package main

import (
    "encoding/json"
    "fmt"
    
    "github.com/spf13/cobra"
)

func NewExportCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "export [id]",
        Short: "Export record(s) as ipset rule",
        Long:  `Export one or all records as ipset rules`,
        Args:  cobra.MaximumNArgs(1),
        Run:   runExport,
    }
    
    return cmd
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