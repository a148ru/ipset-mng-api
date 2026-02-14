// cmd/cli/http.go
package main

import (
    "bytes"
    "fmt"
    "io"
    "net/http"
    "crypto/tls"
)

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
    
    // Настройка TLS если нужно
    if config.Insecure {
        client.Transport = &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
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