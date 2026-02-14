// cmd/cli/auth.go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    //"time"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

func NewLoginCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "login [api-key]",
        Short: "Login and get JWT token",
        Args:  cobra.ExactArgs(1),
        Run:   runLogin,
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
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            viper.SafeWriteConfig()
        } else {
            fmt.Printf("Warning: Failed to save token: %v\n", err)
        }
    }
}