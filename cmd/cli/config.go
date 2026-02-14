// cmd/cli/config.go
package main

import (
    "fmt"
    "os"
    //"path/filepath"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "gopkg.in/yaml.v3"
)

type Config struct {
    APIURL   string `mapstructure:"api_url"`
    Token    string `mapstructure:"token"`
    Output   string `mapstructure:"output"`
    Insecure bool   `mapstructure:"insecure"`
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

func NewConfigCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage configuration",
    }
    
    cmd.AddCommand(&cobra.Command{
        Use:   "set [key] [value]",
        Short: "Set configuration value",
        Args:  cobra.ExactArgs(2),
        Run:   runConfigSet,
    })
    
    cmd.AddCommand(&cobra.Command{
        Use:   "get [key]",
        Short: "Get configuration value",
        Args:  cobra.ExactArgs(1),
        Run:   runConfigGet,
    })
    
    cmd.AddCommand(&cobra.Command{
        Use:   "view",
        Short: "View all configuration",
        Run:   runConfigView,
    })
    
    return cmd
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