package api

import (
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "ipset-api-server/internal/auth"
    "ipset-api-server/internal/config"
    "ipset-api-server/internal/models"
    "ipset-api-server/internal/storage"
    
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
        // Records endpoints
        authorized.GET("/records", s.getAllRecords)
        authorized.GET("/records/:id", s.getRecordByID)
        authorized.POST("/records", s.createRecord)
        authorized.PUT("/records/:id", s.updateRecord)
        authorized.DELETE("/records/:id", s.deleteRecord)
        authorized.GET("/records/search", s.searchRecords)
        
        // Sets endpoints
        authorized.GET("/sets", s.getAllSets)
        authorized.GET("/sets/:set_name", s.getSetByName)
        authorized.DELETE("/sets/:set_name", s.deleteSet)
        authorized.POST("/sets/import", s.importSet)
        authorized.GET("/sets/:set_name/export", s.exportSet)
    }
    
    // Выводим все зарегистрированные маршруты для отладки
    fmt.Println("Registered routes:")
    for _, route := range s.router.Routes() {
        fmt.Printf("  %s %s\n", route.Method, route.Path)
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
        
        token = strings.TrimPrefix(token, "Bearer ")
        
        apiKey, err := s.authManager.ValidateToken(token, s.config.JWTSecret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token"})
            c.Abort()
            return
        }
        
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
        SetName:     req.SetName,
        SetType:     req.SetType,
        SetOptions:  req.SetOptions,
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
    
    existing, err := s.ipsetStorage.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    if req.SetName != "" {
        existing.SetName = req.SetName
    }
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
    if req.SetType != "" {
        existing.SetType = req.SetType
    }
    if req.SetOptions != "" {
        existing.SetOptions = req.SetOptions
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

// Sets endpoints
func (s *Server) getAllSets(c *gin.Context) {
    sets, err := s.ipsetStorage.GetAllSets()
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, sets)
}

func (s *Server) getSetByName(c *gin.Context) {
    setName := c.Param("set_name")
    
    records, err := s.ipsetStorage.GetBySetName(setName)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    if len(records) == 0 {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "set not found"})
        return
    }
    
    // Преобразуем []*models.IPSetRecord в []models.IPSetRecord
    recordList := make([]models.IPSetRecord, len(records))
    for i, r := range records {
        recordList[i] = *r
    }
    
    set := &models.IPSetSet{
        Name:      setName,
        Type:      records[0].SetType,
        Options:   records[0].SetOptions,
        Records:   recordList,
        CreatedAt: records[0].CreatedAt,
        UpdatedAt: records[0].UpdatedAt,
    }
    
    c.JSON(http.StatusOK, set)
}

func (s *Server) deleteSet(c *gin.Context) {
    setName := c.Param("set_name")
    
    if err := s.ipsetStorage.DeleteSet(setName); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{Message: "set deleted successfully"})
}

func (s *Server) importSet(c *gin.Context) {
    var importData struct {
        SetName    string   `json:"set_name" binding:"required"`
        SetType    string   `json:"set_type" binding:"required"`
        SetOptions string   `json:"set_options"`
        Records    []struct {
            IP       string `json:"ip" binding:"required"`
            CIDR     string `json:"cidr"`
            Port     int    `json:"port"`
            Protocol string `json:"protocol"`
        } `json:"records" binding:"required"`
        Description string `json:"description"`
        Context     string `json:"context" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&importData); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    var results []models.ImportResult
    var successCount int
    
    for _, rec := range importData.Records {
        record := &models.IPSetRecord{
            SetName:     importData.SetName,
            SetType:     importData.SetType,
            SetOptions:  importData.SetOptions,
            IP:          rec.IP,
            CIDR:        rec.CIDR,
            Port:        rec.Port,
            Protocol:    rec.Protocol,
            Description: importData.Description,
            Context:     importData.Context,
        }
        
        if err := s.ipsetStorage.Create(record); err != nil {
            results = append(results, models.ImportResult{
                SetName: importData.SetName,
                Records: 0,
                SetType: importData.SetType,
                Success: false,
                Error:   err.Error(),
            })
        } else {
            successCount++
        }
    }
    
    if successCount > 0 {
        results = append(results, models.ImportResult{
            SetName: importData.SetName,
            Records: successCount,
            SetType: importData.SetType,
            Success: true,
        })
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":       "import completed",
        "results":       results,
        "total_success": successCount,
    })
}

func (s *Server) exportSet(c *gin.Context) {
    setName := c.Param("set_name")
    format := c.DefaultQuery("format", "ipset")
    
    records, err := s.ipsetStorage.GetBySetName(setName)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    if len(records) == 0 {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "set not found"})
        return
    }
    
    switch format {
    case "json":
        c.JSON(http.StatusOK, records)
    case "yaml":
        // TODO: implement YAML export
        c.JSON(http.StatusOK, records)
    case "ipset":
        fallthrough
    default:
        export := generateIPSetExport(records)
        c.String(http.StatusOK, export)
    }
}

func generateIPSetExport(records []*models.IPSetRecord) string {
    if len(records) == 0 {
        return ""
    }
    
    var sb strings.Builder
    
    sb.WriteString("#!/bin/bash\n")
    sb.WriteString("# IPSet rules exported from API\n\n")
    
    setMap := make(map[string][]*models.IPSetRecord)
    for _, record := range records {
        setMap[record.SetName] = append(setMap[record.SetName], record)
    }
    
    for setName, setRecords := range setMap {
        setType := "hash:ip"
        setOptions := ""
        if len(setRecords) > 0 {
            if setRecords[0].SetType != "" {
                setType = setRecords[0].SetType
            }
            setOptions = setRecords[0].SetOptions
        }
        
        sb.WriteString(fmt.Sprintf("# Create set: %s\n", setName))
        sb.WriteString(fmt.Sprintf("ipset create %s %s %s -exist\n", 
            setName, setType, setOptions))
        
        for _, record := range setRecords {
            entry := record.IP
            if record.CIDR != "" && record.CIDR != "0" && record.CIDR != "32" {
                entry += "/" + record.CIDR
            }
            
            if record.Port != 0 {
                if record.Protocol != "" {
                    entry += fmt.Sprintf(",%s:%d", record.Protocol, record.Port)
                } else {
                    entry += fmt.Sprintf(",%d", record.Port)
                }
            }
            
            comment := fmt.Sprintf("# %s", record.Description)
            if record.Context != "" {
                comment += fmt.Sprintf(" [%s]", record.Context)
            }
            
            sb.WriteString(fmt.Sprintf("%s\n", comment))
            sb.WriteString(fmt.Sprintf("ipset add %s %s -exist\n", setName, entry))
        }
        sb.WriteString("\n")
    }
    
    sb.WriteString("# Example iptables rules:\n")
    for setName := range setMap {
        sb.WriteString(fmt.Sprintf("# iptables -A INPUT -m set --match-set %s src -j ACCEPT\n", setName))
    }
    
    return sb.String()
}

func (s *Server) Run(addr string) error {
    return s.router.Run(addr)
}

