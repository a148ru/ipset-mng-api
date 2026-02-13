// internal/models/models.go
package models

import (
    "time"
)

type AuthKey struct {
    Key       string    `json:"key"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
    IsActive  bool      `json:"is_active"`
}

type IPSetRecord struct {
    ID          int       `json:"id"`
    IP          string    `json:"ip"`
    CIDR        string    `json:"cidr,omitempty"`
    Port        int       `json:"port,omitempty"`
    Protocol    string    `json:"protocol,omitempty"`
    Description string    `json:"description"`
    Context     string    `json:"context"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type CreateIPSetRequest struct {
    IP          string `json:"ip" binding:"required"`
    CIDR        string `json:"cidr"`
    Port        int    `json:"port"`
    Protocol    string `json:"protocol"`
    Description string `json:"description"`
    Context     string `json:"context" binding:"required"`
}

type UpdateIPSetRequest struct {
    IP          string `json:"ip"`
    CIDR        string `json:"cidr"`
    Port        int    `json:"port"`
    Protocol    string `json:"protocol"`
    Description string `json:"description"`
    Context     string `json:"context"`
}

type LoginRequest struct {
    APIKey string `json:"api_key" binding:"required"`
}

type LoginResponse struct {
    Token string `json:"token"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

type SuccessResponse struct {
    Message string `json:"message"`
}