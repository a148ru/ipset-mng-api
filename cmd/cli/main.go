// cmd/cli/main.go
package main

import (
    "fmt"
    "os"
    "time"
    "net/http"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var config Config
var client *http.Client

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

    // Глобальные флаги
    rootCmd.PersistentFlags().StringVar(&config.APIURL, "api-url", "http://localhost:8080", "API URL")
    rootCmd.PersistentFlags().StringVar(&config.Token, "token", "", "Authentication token")
    rootCmd.PersistentFlags().StringVar(&config.Output, "output", "table", "Output format (json, yaml, table, ipset)")
    rootCmd.PersistentFlags().BoolVar(&config.Insecure, "insecure", false, "Skip TLS verification")
    
    viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
    viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
    viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
    viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))

    // Добавляем все команды
    rootCmd.AddCommand(NewLoginCmd())
    rootCmd.AddCommand(NewRecordsCmd())
    rootCmd.AddCommand(NewSetsCmd())     // Это добавит все команды для работы с сетами
    rootCmd.AddCommand(NewImportCmd())
    rootCmd.AddCommand(NewExportCmd())   // Это для экспорта записей (старая команда)
    rootCmd.AddCommand(NewConfigCmd())

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}