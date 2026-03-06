package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/analyzer"
	"github.com/mungch0120/qsim-cluster/api-server/internal/api/handlers"
	"github.com/mungch0120/qsim-cluster/api-server/internal/api/middleware"
	"github.com/mungch0120/qsim-cluster/api-server/internal/k8s"
	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

// NewRouter creates and configures the API router
func NewRouter(stores *store.Stores, k8sClient *k8s.Client, analyzerClient *analyzer.Client, logger *zap.Logger) *gin.Engine {
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "qsim-api-server",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	v1.Use(middleware.Auth())
	{
		// Initialize handlers
		jobHandler := handlers.NewJobHandler(stores, k8sClient, analyzerClient, logger)
		clusterHandler := handlers.NewClusterHandler(stores, k8sClient, logger)
		analysisHandler := handlers.NewAnalysisHandler(stores, analyzerClient, logger)

		// Job management routes
		jobs := v1.Group("/jobs")
		{
			jobs.POST("", jobHandler.CreateJob)
			jobs.GET("", jobHandler.ListJobs)
			jobs.GET("/:id", jobHandler.GetJob)
			jobs.DELETE("/:id", jobHandler.CancelJob)
			jobs.POST("/:id/retry", jobHandler.RetryJob)
			jobs.GET("/:id/result", jobHandler.GetJobResult)
			jobs.GET("/:id/logs", jobHandler.GetJobLogs)
		}

		// Circuit analysis routes
		analysis := v1.Group("/analyze")
		{
			analysis.POST("", analysisHandler.AnalyzeCircuit)
		}

		// Cluster status routes
		cluster := v1.Group("/cluster")
		{
			cluster.GET("/status", clusterHandler.GetClusterStatus)
			cluster.GET("/nodes", clusterHandler.ListNodes)
			cluster.GET("/metrics", clusterHandler.GetMetrics)
		}
	}

	// WebSocket routes for real-time updates
	router.GET("/ws/jobs/:id", handlers.HandleJobWebSocket(stores, logger))

	return router
}