package auth

import (
    "errors"
    "time"
    "ipset-api-server/internal/storage"
    
    "github.com/golang-jwt/jwt/v5"
)

type Manager struct {
    keyStorage storage.KeyStorage
}

func NewManager(keyStorage storage.KeyStorage) *Manager {
    return &Manager{
        keyStorage: keyStorage,
    }
}

func (m *Manager) ValidateKey(key string) (bool, error) {
    authKey, err := m.keyStorage.GetKey(key)
    if err != nil {
        return false, err
    }
    
    if authKey == nil {
        return false, nil
    }
    
    if !authKey.IsActive {
        return false, nil
    }
    
    if time.Now().After(authKey.ExpiresAt) {
        return false, nil
    }
    
    return true, nil
}

func (m *Manager) GenerateToken(apiKey string, secret string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "api_key": apiKey,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    })
    
    return token.SignedString([]byte(secret))
}

func (m *Manager) ValidateToken(tokenString string, secret string) (string, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(secret), nil
    })
    
    if err != nil {
        return "", err
    }
    
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        apiKey, ok := claims["api_key"].(string)
        if !ok {
            return "", errors.New("invalid token claims")
        }
        return apiKey, nil
    }
    
    return "", errors.New("invalid token")
}

