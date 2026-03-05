package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

// HandleJobWebSocket handles WebSocket connections for real-time job updates
func HandleJobWebSocket(stores *store.Stores, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		
		logger.Info("WebSocket connection requested", zap.String("job_id", jobID))

		// TODO: Implement WebSocket upgrade and real-time job status updates
		// For now, return a placeholder
		c.JSON(501, gin.H{
			"error": "WebSocket not implemented yet",
		})
	}
}