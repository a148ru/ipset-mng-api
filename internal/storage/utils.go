package storage

import "strings"

// contains проверяет, содержит ли строка s подстроку substr
func contains(s, substr string) bool {
    if substr == "" {
        return true
    }
    if s == "" {
        return false
    }
    return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}