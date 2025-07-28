package api

import (
	"net/http"
	"strings"

	"rancher-questions-generator/internal/models"
	"rancher-questions-generator/pkg/helm"
	"rancher-questions-generator/pkg/session"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type Handlers struct {
	sessionManager    *session.Manager
	helmProcessor     *helm.Processor
	repositoryManager *helm.RepositoryManager
}

func NewHandlers() *Handlers {
	return &Handlers{
		sessionManager:    session.NewManager(),
		helmProcessor:     helm.NewProcessor(),
		repositoryManager: helm.NewRepositoryManager(),
	}
}

func (h *Handlers) ProcessChart(c *gin.Context) {
	var req models.ChartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := h.sessionManager.CreateSession(req.URL)

	values, questions, err := h.helmProcessor.ProcessChart(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	session.Values = values
	session.Questions = questions

	response := models.ChartResponse{
		SessionID: session.ID,
		Values:    values,
		Questions: questions,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handlers) GetChart(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	response := models.ChartResponse{
		SessionID: session.ID,
		Values:    session.Values,
		Questions: session.Questions,
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handlers) UpdateChart(c *gin.Context) {
	sessionID := c.Param("session_id")

	var questions models.Questions
	if err := c.ShouldBindJSON(&questions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.sessionManager.UpdateSession(sessionID, questions)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Questions updated successfully"})
}

func (h *Handlers) GetQuestionsYAML(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	yamlData, err := yaml.Marshal(session.Questions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate YAML"})
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", "attachment; filename=questions.yaml")
	c.String(http.StatusOK, string(yamlData))
}

func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// Repository management endpoints

func (h *Handlers) AddRepository(c *gin.Context) {
	var req models.RepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine repository type based on URL
	repoType := "http"
	if strings.HasPrefix(req.URL, "oci://") {
		repoType = "oci"
	}

	err := h.repositoryManager.AddRepositoryWithAuth(req.Name, req.URL, req.Description, repoType, req.Auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Repository added successfully"})
}

func (h *Handlers) ListRepositories(c *gin.Context) {
	repositories := h.repositoryManager.ListRepositories()
	c.JSON(http.StatusOK, gin.H{"repositories": repositories})
}

func (h *Handlers) RemoveRepository(c *gin.Context) {
	name := c.Param("name")
	
	err := h.repositoryManager.RemoveRepository(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Repository removed successfully"})
}

func (h *Handlers) SearchCharts(c *gin.Context) {
	var req models.ChartSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow GET requests with query parameters
		req.Query = c.Query("query")
		req.Repository = c.Query("repository")
	}

	charts, err := h.repositoryManager.SearchCharts(req.Query, req.Repository)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"charts": charts})
}

func (h *Handlers) ProcessChartFromRepository(c *gin.Context) {
	var req models.ChartProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get chart URL from repository
	chartURL, err := h.repositoryManager.PullChart(req.Repository, req.Chart, req.Version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create session
	session := h.sessionManager.CreateSession(chartURL)

	// Process the chart
	values, questions, err := h.helmProcessor.ProcessChart(chartURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	session.Values = values
	session.Questions = questions

	response := models.ChartResponse{
		SessionID: session.ID,
		Values:    values,
		Questions: questions,
	}

	c.JSON(http.StatusOK, response)
}
func (h *Handlers) GetRepositoryCharts(c *gin.Context) {
	repositoryName := c.Param("repository")
	
	charts, err := h.repositoryManager.GetRepositoryCharts(repositoryName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"charts": charts})
}

func (h *Handlers) GetStorageClasses(c *gin.Context) {
	storageClasses, err := h.repositoryManager.GetStorageClasses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"storage_classes": storageClasses})
}
