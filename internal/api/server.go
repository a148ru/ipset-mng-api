package api

import (
    "net/http"
    "strconv"
    "strings"
    "ipset-api-server/internal/auth"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    "ipset-api-server/internal/storage"
    //"ipset-api-server/internal/service"
    
    "github.com/gin-gonic/gin"
)


type Server struct {
    router       *gin.Engine
    config       *config.Config
    authManager  *auth.Manager
    ipsetStorage storage.IPSetStorage
    httpServer   *http.Server
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

// Start запускает HTTP сервер
func (s *Server) Start(addr string) error {
    s.httpServer = &http.Server{
        Addr:    addr,
        Handler: s.router,
    }
    return s.httpServer.ListenAndServe()
}

// Stop останавливает HTTP сервер
func (s *Server) Stop() error {
    if s.httpServer != nil {
        return s.httpServer.Close()
    }
    return nil
}

func (s *Server) setupRoutes() {
    // Публичные маршруты
    s.router.POST("/login", s.login)
    
    // Защищенные маршруты
    authorized := s.router.Group("/")
    authorized.Use(s.authMiddleware())
    {
        // Существующие маршруты для записей
        authorized.GET("/records", s.getAllRecords)
        authorized.GET("/records/:id", s.getRecordByID)
        authorized.POST("/records", s.createRecord)
        authorized.PUT("/records/:id", s.updateRecord)
        authorized.DELETE("/records/:id", s.deleteRecord)
        authorized.GET("/records/search", s.searchRecords)
        
        // Новые маршруты для ipset
        authorized.GET("/ipsets", s.getAllIPSets)
        authorized.POST("/ipsets", s.createIPSet)
        authorized.GET("/ipsets/:name", s.getIPSet)
        authorized.PUT("/ipsets/:name", s.updateIPSet)
        authorized.DELETE("/ipsets/:name", s.deleteIPSet)
        authorized.GET("/ipsets/search", s.searchIPSets)
        
        // Маршруты для записей ipset
        authorized.GET("/ipsets/:name/entries", s.getIPSetEntries)
        authorized.POST("/ipsets/:name/entries", s.addIPSetEntry)
        authorized.DELETE("/ipsets/entries/:entry_id", s.removeIPSetEntry)
        
        // Маршруты для iptables правил
        authorized.GET("/iptables/rules", s.getAllIPTablesRules)
        authorized.POST("/iptables/rules", s.createIPTablesRule)
        authorized.GET("/iptables/rules/:id", s.getIPTablesRule)
        authorized.PUT("/iptables/rules/:id", s.updateIPTablesRule)
        authorized.DELETE("/iptables/rules/:id", s.deleteIPTablesRule)
        authorized.GET("/iptables/rules/search", s.searchIPTablesRules)
        
        // Маршруты для применения конфигурации
        authorized.POST("/apply", s.applyConfiguration)
        authorized.POST("/import", s.importConfiguration)
        
        // Маршруты для генерации команд
        authorized.GET("/generate/ipset/:name", s.generateIPSetCommands)
        authorized.GET("/generate/iptables/:id", s.generateIPTablesCommand)
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

// IPSetRecord handlers
func (s *Server) getAllRecords(c *gin.Context) {
    records, err := s.ipsetStorage.GetAllRecords()
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
    
    record, err := s.ipsetStorage.GetRecordByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, record)
}

func (s *Server) createRecord(c *gin.Context) {
    var req models.CreateIPSetRecordRequest // Изменено с CreateIPSetRequest на CreateIPSetRecordRequest
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
    
    if err := s.ipsetStorage.CreateRecord(record); err != nil {
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
    
    var req models.UpdateIPSetRecordRequest // Изменено с UpdateIPSetRequest на UpdateIPSetRecordRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    // Получаем существующую запись
    existing, err := s.ipsetStorage.GetRecordByID(id)
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
    
    if err := s.ipsetStorage.UpdateRecord(id, existing); err != nil {
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
    
    if err := s.ipsetStorage.DeleteRecord(id); err != nil {
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
    
    records, err := s.ipsetStorage.SearchRecords(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, records)
}