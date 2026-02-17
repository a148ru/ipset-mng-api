// internal/api/ipset_handlers.go (исправленный)
package api

import (
    "net/http"
    "strconv"
    "strings"
    "ipset-api-server/internal/models"
    "ipset-api-server/internal/service"
    
    "github.com/gin-gonic/gin"
)

// IPSet handlers
func (s *Server) createIPSet(c *gin.Context) {
    var req models.CreateIPSetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    set := &models.IPSet{
        Name:        req.Name,
        Type:        req.Type,
        Family:      req.Family,
        HashSize:    req.HashSize,
        MaxElem:     req.MaxElem,
        Description: req.Description,
        Entries:     []models.IPSetEntry{},
    }
    
    if err := s.ipsetStorage.CreateIPSet(set); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, set)
}

func (s *Server) getIPSet(c *gin.Context) {
    name := c.Param("name")
    
    set, err := s.ipsetStorage.GetIPSet(name)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, set)
}

func (s *Server) getAllIPSets(c *gin.Context) {
    sets, err := s.ipsetStorage.GetAllIPSets()
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, sets)
}

func (s *Server) updateIPSet(c *gin.Context) {
    name := c.Param("name")
    
    var req models.CreateIPSetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    set := &models.IPSet{
        Name:        req.Name,
        Type:        req.Type,
        Family:      req.Family,
        HashSize:    req.HashSize,
        MaxElem:     req.MaxElem,
        Description: req.Description,
    }
    
    if err := s.ipsetStorage.UpdateIPSet(name, set); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, set)
}

func (s *Server) deleteIPSet(c *gin.Context) {
    name := c.Param("name")
    
    if err := s.ipsetStorage.DeleteIPSet(name); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{Message: "ipset deleted successfully"})
}

func (s *Server) addIPSetEntry(c *gin.Context) {
    setName := c.Param("name")
    
    var req models.AddIPSetEntryRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    entry := &models.IPSetEntry{
        Value:   req.Value,
        Comment: req.Comment,
    }
    
    if err := s.ipsetStorage.AddIPSetEntry(setName, entry); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, entry)
}

func (s *Server) removeIPSetEntry(c *gin.Context) {
    entryID, err := strconv.Atoi(c.Param("entry_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid entry ID"})
        return
    }
    
    if err := s.ipsetStorage.RemoveIPSetEntry(entryID); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{Message: "entry removed successfully"})
}

func (s *Server) getIPSetEntries(c *gin.Context) {
    setName := c.Param("name")
    
    entries, err := s.ipsetStorage.GetIPSetEntries(setName)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, entries)
}

func (s *Server) searchIPSets(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "search query required"})
        return
    }
    
    sets, err := s.ipsetStorage.SearchIPSets(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, sets)
}

// IPTables handlers
func (s *Server) createIPTablesRule(c *gin.Context) {
    var req models.CreateIPTablesRuleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    rule := &models.IPTablesRule{
        Chain:       req.Chain,
        Interface:   req.Interface,
        Protocol:    req.Protocol,
        SrcSets:     req.SrcSets,
        DstSets:     req.DstSets,
        Action:      req.Action,
        Description: req.Description,
        Position:    req.Position,
    }
    
    if err := s.ipsetStorage.CreateIPTablesRule(rule); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, rule)
}

func (s *Server) getIPTablesRule(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID"})
        return
    }
    
    rule, err := s.ipsetStorage.GetIPTablesRule(id)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, rule)
}

func (s *Server) getAllIPTablesRules(c *gin.Context) {
    rules, err := s.ipsetStorage.GetAllIPTablesRules()
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, rules)
}

func (s *Server) updateIPTablesRule(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID"})
        return
    }
    
    var req models.CreateIPTablesRuleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    rule := &models.IPTablesRule{
        Chain:       req.Chain,
        Interface:   req.Interface,
        Protocol:    req.Protocol,
        SrcSets:     req.SrcSets,
        DstSets:     req.DstSets,
        Action:      req.Action,
        Description: req.Description,
        Position:    req.Position,
    }
    
    if err := s.ipsetStorage.UpdateIPTablesRule(id, rule); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, rule)
}

func (s *Server) deleteIPTablesRule(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID"})
        return
    }
    
    if err := s.ipsetStorage.DeleteIPTablesRule(id); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{Message: "rule deleted successfully"})
}

func (s *Server) searchIPTablesRules(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "search query required"})
        return
    }
    
    rules, err := s.ipsetStorage.SearchIPTablesRules(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, rules)
}

// Apply configuration
func (s *Server) applyConfiguration(c *gin.Context) {
    var req models.ApplyRulesRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    dryRun := c.Query("dry_run") == "true"
    manager := service.NewIPTablesManager(dryRun)
    
    if err := manager.ApplyConfiguration(req.IPSetCommands, req.IPTablesCommands); err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    message := "configuration applied successfully"
    if dryRun {
        message = "dry run completed - no changes were made"
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{Message: message})
}

// Import configuration from text
func (s *Server) importConfiguration(c *gin.Context) {
    var req struct {
        Config string `json:"config" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    manager := service.NewIPTablesManager(true)
    lines := strings.Split(req.Config, "\n")
    
    var ipsetCommands []models.IPSetCommand
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        if strings.HasPrefix(line, "create ") {
            set, err := manager.ParseIPSetConfig(line)
            if err != nil {
                c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
                return
            }
            if err := s.ipsetStorage.CreateIPSet(set); err != nil {
                c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
                return
            }
            
            ipsetCommands = append(ipsetCommands, models.IPSetCommand{
                Command: "create",
                SetName: set.Name,
                Args:    []string{string(set.Type), "family", string(set.Family), "hashsize", strconv.Itoa(set.HashSize), "maxelem", strconv.Itoa(set.MaxElem)},
            })
        } else if strings.HasPrefix(line, "add ") {
            setName, value, err := manager.ParseIPSetEntry(line)
            if err != nil {
                c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
                return
            }
            
            entry := &models.IPSetEntry{
                Value:   value,
                Comment: "imported",
            }
            
            if err := s.ipsetStorage.AddIPSetEntry(setName, entry); err != nil {
                c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
                return
            }
            
            ipsetCommands = append(ipsetCommands, models.IPSetCommand{
                Command: "add",
                SetName: setName,
                Args:    []string{value},
            })
        } else if strings.HasPrefix(line, "iptables ") {
            // Сохраняем iptables команды как есть
            // В реальном приложении здесь нужно парсить iptables команды и создавать правила
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":        "configuration imported successfully",
        "ipset_commands": ipsetCommands,
    })
}

// Generate commands
func (s *Server) generateIPSetCommands(c *gin.Context) {
    name := c.Param("name")
    
    set, err := s.ipsetStorage.GetIPSet(name)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    manager := service.NewIPTablesManager(true)
    commands := []string{
        manager.GenerateIPSetCommand(set),
    }
    
    for _, entry := range set.Entries {
        commands = append(commands, manager.GenerateIPSetAddCommand(set.Name, &entry))
    }
    
    c.JSON(http.StatusOK, gin.H{"commands": commands})
}

func (s *Server) generateIPTablesCommand(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid ID"})
        return
    }
    
    rule, err := s.ipsetStorage.GetIPTablesRule(id)
    if err != nil {
        c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
        return
    }
    
    manager := service.NewIPTablesManager(true)
    command := manager.GenerateIPTablesRule(rule, "append")
    
    c.JSON(http.StatusOK, gin.H{"command": command})
}