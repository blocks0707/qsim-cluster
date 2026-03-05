package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

type ClusterHandler struct {
	stores *store.Stores
	logger *zap.Logger
}

func NewClusterHandler(stores *store.Stores, logger *zap.Logger) *ClusterHandler {
	return &ClusterHandler{
		stores: stores,
		logger: logger,
	}
}

// GetClusterStatus returns overall cluster status
func (h *ClusterHandler) GetClusterStatus(c *gin.Context) {
	// TODO: Get actual cluster status from Kubernetes
	status := gin.H{
		"status": "healthy",
		"version": "v0.1.0",
		"nodes": gin.H{
			"total": 3,
			"ready": 3,
			"cpu_pool": 2,
			"gpu_pool": 1,
		},
		"jobs": gin.H{
			"total": 0,
			"pending": 0,
			"running": 0,
			"completed": 0,
			"failed": 0,
		},
		"resources": gin.H{
			"cpu_usage": "25%",
			"memory_usage": "40%",
			"gpu_usage": "0%",
		},
	}

	c.JSON(http.StatusOK, status)
}

// ListNodes returns list of nodes and their status
func (h *ClusterHandler) ListNodes(c *gin.Context) {
	// TODO: Get actual node information from Kubernetes
	nodes := []gin.H{
		{
			"name": "worker-01",
			"pool": "cpu",
			"status": "ready",
			"cpu_cores": 8,
			"memory_gb": 32,
			"gpu": false,
			"active_jobs": 0,
			"cpu_usage": "20%",
			"memory_usage": "35%",
		},
		{
			"name": "worker-02",
			"pool": "cpu",
			"status": "ready",
			"cpu_cores": 8,
			"memory_gb": 32,
			"gpu": false,
			"active_jobs": 1,
			"cpu_usage": "45%",
			"memory_usage": "60%",
		},
		{
			"name": "worker-03",
			"pool": "gpu",
			"status": "ready",
			"cpu_cores": 64,
			"memory_gb": 256,
			"gpu": true,
			"gpu_type": "A100",
			"active_jobs": 0,
			"cpu_usage": "10%",
			"memory_usage": "15%",
			"gpu_usage": "0%",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"total": len(nodes),
	})
}

// GetMetrics returns cluster metrics for monitoring
func (h *ClusterHandler) GetMetrics(c *gin.Context) {
	// TODO: Get actual metrics from Prometheus
	metrics := gin.H{
		"timestamp": "2024-01-15T10:30:00Z",
		"cluster": gin.H{
			"total_cpu_cores": 80,
			"total_memory_gb": 320,
			"total_gpus": 1,
			"cpu_utilization": 0.25,
			"memory_utilization": 0.40,
			"gpu_utilization": 0.0,
		},
		"jobs": gin.H{
			"jobs_per_minute": 2.5,
			"avg_execution_time_sec": 120,
			"success_rate": 0.95,
			"queue_depth": 3,
		},
		"complexity_distribution": gin.H{
			"class_a": 60,
			"class_b": 30,
			"class_c": 8,
			"class_d": 2,
		},
	}

	c.JSON(http.StatusOK, metrics)
}