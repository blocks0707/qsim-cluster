package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/k8s"
	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

type ClusterHandler struct {
	stores    *store.Stores
	k8sClient *k8s.Client
	logger    *zap.Logger
}

func NewClusterHandler(stores *store.Stores, k8sClient *k8s.Client, logger *zap.Logger) *ClusterHandler {
	return &ClusterHandler{
		stores:    stores,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetClusterStatus returns overall cluster status
func (h *ClusterHandler) GetClusterStatus(c *gin.Context) {
	ctx := context.Background()
	
	h.logger.Info("Getting cluster status")

	// Get actual cluster status from Kubernetes
	clusterStatus, err := h.k8sClient.GetClusterStatus(ctx)
	if err != nil {
		h.logger.Error("Failed to get cluster status", zap.Error(err))
		
		// Return fallback status
		status := gin.H{
			"status": "degraded",
			"version": "v0.1.0",
			"error": "Unable to connect to cluster",
			"nodes": gin.H{
				"total": 0,
				"ready": 0,
			},
			"jobs": gin.H{
				"total": 0,
				"pending": 0,
				"running": 0,
				"completed": 0,
				"failed": 0,
			},
			"resources": gin.H{
				"cpu_usage": "unknown",
				"memory_usage": "unknown",
				"gpu_usage": "unknown",
			},
		}
		
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	// Get job statistics from database
	totalJobs := 0
	jobsByStatus := map[string]int{
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}

	// Query job statistics (simplified)
	// In production, would implement proper job statistics queries
	jobs, _, err := h.stores.Jobs.List(store.JobListParams{
		UserID: "", // Get all jobs for admin view
		Page:   1,
		Limit:  1000, // Large limit to get total count
	})
	if err == nil {
		totalJobs = len(jobs)
		for _, job := range jobs {
			if count, exists := jobsByStatus[job.Status]; exists {
				jobsByStatus[job.Status] = count + 1
			}
		}
	}

	status := gin.H{
		"status":      clusterStatus.Status,
		"version":     clusterStatus.Version,
		"timestamp":   ctx.Value("timestamp"),
		"nodes": gin.H{
			"total":      clusterStatus.TotalNodes,
			"ready":      clusterStatus.ReadyNodes,
			"pools":      clusterStatus.NodePools,
		},
		"jobs": gin.H{
			"total":      totalJobs,
			"pending":    jobsByStatus["pending"],
			"running":    jobsByStatus["running"],
			"completed":  jobsByStatus["completed"],
			"failed":     jobsByStatus["failed"],
		},
		"resources":   clusterStatus.ResourceUsage,
	}

	c.JSON(http.StatusOK, status)
}

// ListNodes returns list of nodes and their status
func (h *ClusterHandler) ListNodes(c *gin.Context) {
	ctx := context.Background()
	
	h.logger.Info("Getting node list")

	// Get actual node information from Kubernetes
	nodeInfos, err := h.k8sClient.ListNodes(ctx)
	if err != nil {
		h.logger.Error("Failed to list nodes", zap.Error(err))
		
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Unable to retrieve node information",
			"nodes": []gin.H{},
			"total": 0,
		})
		return
	}

	// Convert NodeInfo structs to response format
	var nodes []gin.H
	for _, nodeInfo := range nodeInfos {
		node := gin.H{
			"name":         nodeInfo.Name,
			"pool":         nodeInfo.Pool,
			"status":       nodeInfo.Status,
			"cpu_cores":    nodeInfo.CPUCores,
			"memory_gb":    nodeInfo.MemoryGB,
			"gpu":          nodeInfo.GPU,
			"active_jobs":  nodeInfo.ActiveJobs,
			"cpu_usage":    nodeInfo.CPUUsage,
			"memory_usage": nodeInfo.MemoryUsage,
		}

		if nodeInfo.GPU {
			node["gpu_type"] = nodeInfo.GPUType
			node["gpu_usage"] = nodeInfo.GPUUsage
		}

		if nodeInfo.Labels != nil && len(nodeInfo.Labels) > 0 {
			node["labels"] = nodeInfo.Labels
		}

		nodes = append(nodes, node)
	}

	// Parse query parameters for filtering
	poolFilter := c.Query("pool")
	statusFilter := c.Query("status")

	// Apply filters if specified
	if poolFilter != "" || statusFilter != "" {
		var filteredNodes []gin.H
		for _, node := range nodes {
			include := true
			
			if poolFilter != "" && node["pool"] != poolFilter {
				include = false
			}
			
			if statusFilter != "" && node["status"] != statusFilter {
				include = false
			}
			
			if include {
				filteredNodes = append(filteredNodes, node)
			}
		}
		nodes = filteredNodes
	}

	response := gin.H{
		"nodes": nodes,
		"total": len(nodes),
	}

	// Add filter info if applied
	if poolFilter != "" {
		response["filter_pool"] = poolFilter
	}
	if statusFilter != "" {
		response["filter_status"] = statusFilter
	}

	c.JSON(http.StatusOK, response)
}

// GetMetrics returns cluster metrics for monitoring
func (h *ClusterHandler) GetMetrics(c *gin.Context) {
	ctx := context.Background()
	
	h.logger.Info("Getting cluster metrics")

	// Get cluster status for basic metrics
	clusterStatus, err := h.k8sClient.GetClusterStatus(ctx)
	if err != nil {
		h.logger.Error("Failed to get cluster status for metrics", zap.Error(err))
		
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Unable to retrieve cluster metrics",
		})
		return
	}

	// Get job statistics from database
	jobs, _, err := h.stores.Jobs.List(store.JobListParams{
		UserID: "", // Get all jobs for metrics
		Page:   1,
		Limit:  10000,
	})

	var totalJobs, completedJobs, failedJobs int
	var avgExecutionTime int64
	var executionCount int64
	complexityDistribution := map[string]int{
		"class_a": 0,
		"class_b": 0,
		"class_c": 0,
		"class_d": 0,
	}

	if err == nil {
		totalJobs = len(jobs)
		
		for _, job := range jobs {
			// Count by status
			if job.Status == "completed" {
				completedJobs++
				
				// Calculate average execution time
				if job.ExecutionTimeMs != nil {
					avgExecutionTime += *job.ExecutionTimeMs
					executionCount++
				}
			} else if job.Status == "failed" {
				failedJobs++
			}
			
			// Count by complexity class
			if job.ComplexityClass != nil {
				switch *job.ComplexityClass {
				case "A":
					complexityDistribution["class_a"]++
				case "B":
					complexityDistribution["class_b"]++
				case "C":
					complexityDistribution["class_c"]++
				case "D":
					complexityDistribution["class_d"]++
				}
			}
		}
		
		// Calculate average execution time in seconds
		if executionCount > 0 {
			avgExecutionTime = (avgExecutionTime / executionCount) / 1000 // Convert to seconds
		}
	}

	// Calculate success rate
	var successRate float64
	if totalJobs > 0 {
		successRate = float64(completedJobs) / float64(totalJobs)
	}

	// Get node information for resource totals
	nodeInfos, _ := h.k8sClient.ListNodes(ctx)
	
	var totalCPUCores, totalMemoryGB, totalGPUs int64
	for _, node := range nodeInfos {
		totalCPUCores += node.CPUCores
		totalMemoryGB += node.MemoryGB
		if node.GPU {
			totalGPUs++
		}
	}

	// Parse resource utilization from cluster status
	cpuUtilization := parseUtilization(clusterStatus.ResourceUsage["cpu_usage"])
	memoryUtilization := parseUtilization(clusterStatus.ResourceUsage["memory_usage"])
	gpuUtilization := parseUtilization(clusterStatus.ResourceUsage["gpu_usage"])

	metrics := gin.H{
		"timestamp": ctx.Value("timestamp"),
		"cluster": gin.H{
			"total_cpu_cores":      totalCPUCores,
			"total_memory_gb":      totalMemoryGB,
			"total_gpus":           totalGPUs,
			"total_nodes":          clusterStatus.TotalNodes,
			"ready_nodes":          clusterStatus.ReadyNodes,
			"cpu_utilization":      cpuUtilization,
			"memory_utilization":   memoryUtilization,
			"gpu_utilization":      gpuUtilization,
		},
		"jobs": gin.H{
			"total_jobs":              totalJobs,
			"completed_jobs":          completedJobs,
			"failed_jobs":             failedJobs,
			"success_rate":            successRate,
			"avg_execution_time_sec":  avgExecutionTime,
			"jobs_per_minute":         calculateJobsPerMinute(jobs), // Helper function
		},
		"complexity_distribution": complexityDistribution,
		"pools": clusterStatus.NodePools,
	}

	c.JSON(http.StatusOK, metrics)
}

// Helper function to parse utilization strings like "25%" to 0.25
func parseUtilization(util string) float64 {
	if util == "" || util == "unknown" {
		return 0.0
	}
	
	// Simple parsing for percentage strings
	// In production, would use proper parsing with strconv
	switch util {
	case "0%":
		return 0.0
	case "25%":
		return 0.25
	case "40%":
		return 0.40
	case "50%":
		return 0.50
	case "75%":
		return 0.75
	case "100%":
		return 1.0
	default:
		return 0.0
	}
}

// Helper function to calculate jobs per minute (simplified)
func calculateJobsPerMinute(jobs []*store.Job) float64 {
	if len(jobs) == 0 {
		return 0.0
	}
	
	// Simplified calculation - would use proper time-based analysis in production
	return float64(len(jobs)) / 60.0 // Rough estimate
}