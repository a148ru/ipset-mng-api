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
    SetName     string    `json:"set_name"`
    IP          string    `json:"ip"`
    CIDR        string    `json:"cidr,omitempty"`
    Port        int       `json:"port,omitempty"`
    Protocol    string    `json:"protocol,omitempty"`
    Description string    `json:"description"`
    Context     string    `json:"context"`
    SetType     string    `json:"set_type,omitempty"`
    SetOptions  string    `json:"set_options,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type IPSetSet struct {
    Name        string         `json:"name"`
    Type        string         `json:"type"`
    Options     string         `json:"options,omitempty"`
    Records     []IPSetRecord  `json:"records"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateIPSetRequest struct {
    SetName     string `json:"set_name" binding:"required"`
    IP          string `json:"ip" binding:"required"`
    CIDR        string `json:"cidr"`
    Port        int    `json:"port"`
    Protocol    string `json:"protocol"`
    Description string `json:"description"`
    Context     string `json:"context" binding:"required"`
    SetType     string `json:"set_type"`
    SetOptions  string `json:"set_options"`
}

type UpdateIPSetRequest struct {
    SetName     string `json:"set_name"`
    IP          string `json:"ip"`
    CIDR        string `json:"cidr"`
    Port        int    `json:"port"`
    Protocol    string `json:"protocol"`
    Description string `json:"description"`
    Context     string `json:"context"`
    SetType     string `json:"set_type"`
    SetOptions  string `json:"set_options"`
}

type ImportResult struct {
    SetName     string   `json:"set_name"`
    Records     int      `json:"records"`
    SetType     string   `json:"set_type"`
    Success     bool     `json:"success"`
    Error       string   `json:"error,omitempty"`
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

