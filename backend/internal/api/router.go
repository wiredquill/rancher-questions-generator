package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	handlers := NewHandlers()

	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
		
		// Legacy chart processing (direct URL)
		api.POST("/chart", handlers.ProcessChart)
		api.GET("/chart/:session_id", handlers.GetChart)
		api.PUT("/chart/:session_id", handlers.UpdateChart)
		api.GET("/chart/:session_id/q", handlers.GetQuestionsYAML)
		
		// Repository management
		api.POST("/repositories", handlers.AddRepository)
		api.GET("/repositories", handlers.ListRepositories)
		api.DELETE("/repositories/:name", handlers.RemoveRepository)
		
		// Chart search and processing from repositories
		api.GET("/charts/search", handlers.SearchCharts)
		api.POST("/charts/search", handlers.SearchCharts)
		api.POST("/charts/process", handlers.ProcessChartFromRepository)
		api.GET("/repositories/:repository/charts", handlers.GetRepositoryCharts)
		
		// System information
		api.GET("/storage-classes", handlers.GetStorageClasses)
	}

	return router
}