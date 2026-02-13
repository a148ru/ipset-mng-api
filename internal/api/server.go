// internal/api/server.go
package api

import (
    "net/http"
    "strconv"
    "strings"
    "ipset-api/internal/auth"
    "ipset-api/internal/config"
    "ipset-api/internal/models"
    "ipset-api/internal/storage"
    
    "github.com/gin-gonic/gin"
)

type Server struct {
    router       *gin.Engine
    config       *config.Config
    authManager  *auth.Manager
    ipsetStorage storage.IPSetStorage
}

func NewServer(cfg *config.Config, authManager *auth.Manager, ipsetStorage storage.IPSetStorage) *Server {
    server := &Server{
        router:       gin.Default(),
        config:       cfg,
        authManager:  authManager,
        ipsetStorage: ipsetStorage,
    }
    
    server.setupRoutes()
    return server
}

func (s *Server) setupRoutes() {
    // Публичные маршруты
    s.router.POST("/login", s.login)
    
    // Защищенные маршруты
    authorized := s.router.Group("/")
    authorized.Use(s.authMiddleware())
    {
        authorized.GET("/records", s.getAllRecords)
        authorized.GET("/records/:id", s.getRecordByID)
        authorized.POST("/records", s.createRecord)
        authorized.PUT("/records/:id", s.updateRecord)
        authorized.DELETE("/records/:id", s.deleteRecord)
        authorized.GET("/records/search", s.searchRecords)
    }
}

func (s *Server) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authorization token required"})
            c.Abort()
            return
        }
        
        // Удаляем префикс "Bearer " если есть
        token = strings.TrimPrefix(token, "Bearer ")
        
        apiKey, err := s.authManager.ValidateToken(token, s.config.JWTSecret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token"})
            c.Abort()
            return
        }
        
        // Проверяем что ключ все еще действителен
        valid, err := s.authManager.ValidateKey(apiKey)
        if err != nil || !valid {
            c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid or expired API key"})
            c.Abort()
            return
        }
        
        c.Set("api_key", apiKey)
        c.Next()
    }
}

func (s *Server) login(c *gin.Context) {
    var req models.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request"})
        return
    }
    
    valid, err := s.authManager.ValidateKey(req.APIKey)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal server error"})
        return
    }
    
    if !valid {
        c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid API key"})
        return
    }
    
    token, err := s.authManager.GenerateToken(req.APIKey, s.config.JWTSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate token"})
        return
    }
    
    c.JSON(http.StatusOK, models.LoginResponse{Token: token})
}

func (s *Server) getAllRecords(c *gin.Context) {
    records, err := s.ipsetStorage.GetAll()
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, records)
}

func (s *Server) getRecordByID(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id < 100000 || id > 999999 {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID (must be 6-digit number)"})
        return
    }
    
    record, err := s.ipsetStorage.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, record)
}

func (s *Server) createRecord(c *gin.Context) {
    var req models.CreateIPSetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    record := &models.IPSetRecord{
        IP:          req.IP,
        CIDR:        req.CIDR,
        Port:        req.Port,
        Protocol:    req.Protocol,
        Description: req.Description,
        Context:     req.Context,
    }
    
    if err := s.ipsetStorage.Create(record); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, record)
}

func (s *Server) updateRecord(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id < 100000 || id > 999999 {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID (must be 6-digit number)"})
        return
    }
    
    var req models.UpdateIPSetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    // Получаем существующую запись
    existing, err := s.ipsetStorage.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    // Обновляем поля
    if req.IP != "" {
        existing.IP = req.IP
    }
    if req.CIDR != "" {
        existing.CIDR = req.CIDR
    }
    if req.Port != 0 {
        existing.Port = req.Port
    }
    if req.Protocol != "" {
        existing.Protocol = req.Protocol
    }
    if req.Description != "" {
        existing.Description = req.Description
    }
    if req.Context != "" {
        existing.Context = req.Context
    }
    
    if err := s.ipsetStorage.Update(id, existing); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, existing)
}

func (s *Server) deleteRecord(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id < 100000 || id > 999999 {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID (must be 6-digit number)"})
        return
    }
    
    if err := s.ipsetStorage.Delete(id); err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{Message: "record deleted successfully"})
}

func (s *Server) searchRecords(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "search query required"})
        return
    }
    
    records, err := s.ipsetStorage.Search(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, records)
}

func (s *Server) Run(addr string) error {
    return s.router.Run(addr)
}