// cmd/cli/records.go
package main

import (
    "encoding/json"
    "fmt"
    //"strconv"
    
    "github.com/spf13/cobra"
)

func NewRecordsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "records",
        Short: "Manage IPSet records",
    }

    cmd.AddCommand(NewListRecordsCmd())
    cmd.AddCommand(NewGetRecordCmd())
    cmd.AddCommand(NewCreateRecordCmd())
    cmd.AddCommand(NewUpdateRecordCmd())
    cmd.AddCommand(NewDeleteRecordCmd())
    cmd.AddCommand(NewSearchRecordsCmd())

    return cmd
}

func NewListRecordsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all records",
        Run:   runListRecords,
    }
}

func NewGetRecordCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get [id]",
        Short: "Get record by ID",
        Args:  cobra.ExactArgs(1),
        Run:   runGetRecord,
    }
}

func NewCreateRecordCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "create",
        Short: "Create a new record",
        Run:   runCreateRecord,
    }
    
    cmd.Flags().StringP("set-name", "s", "", "Set name (required)")
    cmd.Flags().StringP("ip", "i", "", "IP address (required)")
    cmd.Flags().StringP("cidr", "c", "", "CIDR mask")
    cmd.Flags().IntP("port", "p", 0, "Port number")
    cmd.Flags().StringP("protocol", "r", "", "Protocol (tcp/udp)")
    cmd.Flags().StringP("description", "d", "", "Description")
    cmd.Flags().StringP("context", "x", "", "Context (required)")
    cmd.Flags().StringP("set-type", "t", "hash:ip", "Set type")
    cmd.Flags().StringP("set-options", "o", "", "Set options")
    
    cmd.MarkFlagRequired("set-name")
    cmd.MarkFlagRequired("ip")
    cmd.MarkFlagRequired("context")
    
    return cmd
}

func NewUpdateRecordCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "update [id]",
        Short: "Update an existing record",
        Args:  cobra.ExactArgs(1),
        Run:   runUpdateRecord,
    }
    
    cmd.Flags().StringP("set-name", "s", "", "Set name")
    cmd.Flags().StringP("ip", "i", "", "IP address")
    cmd.Flags().StringP("cidr", "c", "", "CIDR mask")
    cmd.Flags().IntP("port", "p", 0, "Port number")
    cmd.Flags().StringP("protocol", "r", "", "Protocol (tcp/udp)")
    cmd.Flags().StringP("description", "d", "", "Description")
    cmd.Flags().StringP("context", "x", "", "Context")
    cmd.Flags().StringP("set-type", "t", "", "Set type")
    cmd.Flags().StringP("set-options", "o", "", "Set options")
    
    return cmd
}

func NewDeleteRecordCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "delete [id]",
        Short: "Delete a record",
        Args:  cobra.ExactArgs(1),
        Run:   runDeleteRecord,
    }
}

func NewSearchRecordsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "search [query]",
        Short: "Search records by context",
        Args:  cobra.ExactArgs(1),
        Run:   runSearchRecords,
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
    setName, _ := cmd.Flags().GetString("set-name")
    ip, _ := cmd.Flags().GetString("ip")
    cidr, _ := cmd.Flags().GetString("cidr")
    port, _ := cmd.Flags().GetInt("port")
    protocol, _ := cmd.Flags().GetString("protocol")
    description, _ := cmd.Flags().GetString("description")
    context, _ := cmd.Flags().GetString("context")
    setType, _ := cmd.Flags().GetString("set-type")
    setOptions, _ := cmd.Flags().GetString("set-options")

    record := map[string]interface{}{
        "set_name": setName,
        "ip":       ip,
        "context":  context,
        "set_type": setType,
    }
    
    if setOptions != "" {
        record["set_options"] = setOptions
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
    
    if setName, _ := cmd.Flags().GetString("set-name"); setName != "" {
        record["set_name"] = setName
    }
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
    if setType, _ := cmd.Flags().GetString("set-type"); setType != "" {
        record["set_type"] = setType
    }
    if setOptions, _ := cmd.Flags().GetString("set-options"); setOptions != "" {
        record["set_options"] = setOptions
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