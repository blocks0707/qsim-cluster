package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/analyzer"
	"github.com/mungch0120/qsim-cluster/api-server/internal/k8s"
	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

type JobHandler struct {
	stores          *store.Stores
	k8sClient       *k8s.Client
	analyzerClient  *analyzer.Client
	logger          *zap.Logger
}

func NewJobHandler(stores *store.Stores, k8sClient *k8s.Client, analyzerClient *analyzer.Client, logger *zap.Logger) *JobHandler {
	return &JobHandler{
		stores:         stores,
		k8sClient:      k8sClient,
		analyzerClient: analyzerClient,
		logger:         logger,
	}
}

// CreateJobRequest represents the request body for job creation
type CreateJobRequest struct {
	Code     string                 `json:"code" binding:"required"`
	Language string                 `json:"language,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// JobResponse represents the job response
type JobResponse struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	Status         string                 `json:"status"`
	Code           string                 `json:"code,omitempty"`
	Language       string                 `json:"language,omitempty"`
	Complexity     map[string]interface{} `json:"complexity,omitempty"`
	AssignedNode   string                 `json:"assigned_node,omitempty"`
	AssignedPool   string                 `json:"assigned_pool,omitempty"`
	StartTime      *string                `json:"start_time,omitempty"`
	CompletionTime *string                `json:"completion_time,omitempty"`
	ExecutionTime  *int                   `json:"execution_time,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

// CreateJob creates a new quantum job
func (h *JobHandler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Set default language if not specified
	if req.Language == "" {
		req.Language = "python"
	}

	h.logger.Info("Creating quantum job",
		zap.String("job_id", jobID),
		zap.String("user_id", userID.(string)),
		zap.String("language", req.Language),
	)

	// Step 1: Analyze circuit complexity
	ctx := context.Background()
	analysisReq := &analyzer.AnalyzeRequest{
		Code:     req.Code,
		Language: req.Language,
	}

	analysis, err := h.analyzerClient.Analyze(ctx, analysisReq)
	if err != nil {
		h.logger.Warn("Circuit analyzer failed, using fallback estimation", zap.Error(err))
		// Use fallback estimation
		analysis = analyzer.EstimateResources(req.Code, req.Language)
	}

	h.logger.Info("Circuit analysis completed",
		zap.String("job_id", jobID),
		zap.Int("qubits", analysis.Qubits),
		zap.String("complexity_class", analysis.ComplexityClass),
		zap.String("recommended_pool", analysis.RecommendedPool),
	)

	// Step 2: Create job record in database
	job := &store.Job{
		ID:              jobID,
		UserID:          userID.(string),
		Status:          "analyzing",
		Code:            req.Code,
		Language:        req.Language,
		Qubits:          &analysis.Qubits,
		Depth:           &analysis.Depth,
		GateCount:       &analysis.GateCount,
		ComplexityClass: &analysis.ComplexityClass,
		Method:          &analysis.RecommendedMethod,
		Priority:        "normal",
	}

	if err := h.stores.Jobs.Create(job); err != nil {
		h.logger.Error("Failed to create job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create job",
		})
		return
	}

	// Step 3: Update job complexity in database
	err = h.stores.Jobs.UpdateComplexity(jobID, analysis.Qubits, analysis.Depth, 
		analysis.GateCount, analysis.ComplexityClass, analysis.RecommendedMethod)
	if err != nil {
		h.logger.Warn("Failed to update job complexity", zap.Error(err))
	}

	// Step 4: Create QuantumJob CR in Kubernetes
	quantumJob := &k8s.QuantumJob{
		ID:       jobID,
		UserID:   userID.(string),
		Code:     req.Code,
		Language: req.Language,
		Complexity: map[string]interface{}{
			"qubits":             analysis.Qubits,
			"depth":              analysis.Depth,
			"gateCount":          analysis.GateCount,
			"class":              analysis.ComplexityClass,
			"parallelism":        analysis.Parallelism,
			"estimatedMemoryMB":  analysis.EstimatedMemoryMB,
			"estimatedCPUCores":  analysis.EstimatedCPU,
			"estimatedTimeSec":   analysis.EstimatedTimeSec,
			"method":             analysis.RecommendedMethod,
		},
		Scheduling: map[string]interface{}{
			"priority":   "normal",
			"nodePool":   analysis.RecommendedPool,
			"timeout":    300,
			"retryPolicy": map[string]interface{}{
				"maxRetries":      2,
				"backoffSeconds":  30,
			},
		},
		Resources: map[string]string{
			"cpu":    fmt.Sprintf("%d", analysis.EstimatedCPU),
			"memory": fmt.Sprintf("%dMi", analysis.EstimatedMemoryMB),
			"gpu":    "0",
		},
	}

	err = h.k8sClient.CreateQuantumJob(ctx, quantumJob)
	if err != nil {
		h.logger.Error("Failed to create QuantumJob CR", zap.Error(err))
		// Update job status to failed
		h.stores.Jobs.UpdateStatus(jobID, userID.(string), "failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to submit job to cluster",
		})
		return
	}

	// Update job status to submitted
	err = h.stores.Jobs.UpdateStatus(jobID, userID.(string), "submitted")
	if err != nil {
		h.logger.Warn("Failed to update job status", zap.Error(err))
	}

	c.JSON(http.StatusCreated, gin.H{
		"job_id": jobID,
		"status": "submitted",
		"message": "Job submitted successfully",
		"analysis": gin.H{
			"qubits":              analysis.Qubits,
			"complexity_class":    analysis.ComplexityClass,
			"estimated_time_sec":  analysis.EstimatedTimeSec,
			"recommended_pool":    analysis.RecommendedPool,
		},
	})
}

// ListJobs lists jobs for the authenticated user
func (h *JobHandler) ListJobs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse pagination parameters
	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Parse filters
	status := c.Query("status")

	jobs, total, err := h.stores.Jobs.List(store.JobListParams{
		UserID: userID.(string),
		Status: status,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		h.logger.Error("Failed to list jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list jobs",
		})
		return
	}

	// Convert to response format
	var response []JobResponse
	for _, job := range jobs {
		response = append(response, JobResponse{
			ID:        job.ID,
			UserID:    job.UserID,
			Status:    job.Status,
			Language:  job.Language,
			CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: job.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  response,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetJob retrieves a specific job
func (h *JobHandler) GetJob(c *gin.Context) {
	jobID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	job, err := h.stores.Jobs.GetByID(jobID)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		h.logger.Error("Failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
		})
		return
	}

	// Check if user owns this job
	if job.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	response := JobResponse{
		ID:           job.ID,
		UserID:       job.UserID,
		Status:       job.Status,
		Code:         job.Code,
		Language:     job.Language,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    job.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}

// CancelJob cancels a running or pending job
func (h *JobHandler) CancelJob(c *gin.Context) {
	jobID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	h.logger.Info("Cancelling job", zap.String("job_id", jobID))

	// First, check if the job exists and belongs to the user
	job, err := h.stores.Jobs.GetByID(jobID)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		h.logger.Error("Failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
		})
		return
	}

	if job.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if job can be cancelled
	if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot cancel job in %s state", job.Status),
		})
		return
	}

	// Cancel the QuantumJob CR in Kubernetes
	ctx := context.Background()
	err = h.k8sClient.DeleteQuantumJob(ctx, jobID)
	if err != nil {
		h.logger.Error("Failed to delete QuantumJob CR", zap.Error(err))
		// Continue with database update even if k8s deletion fails
	}

	// Update status in database
	err = h.stores.Jobs.UpdateStatus(jobID, userID.(string), "cancelled")
	if err != nil {
		h.logger.Error("Failed to update job status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cancel job",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancelled successfully",
	})
}

// RetryJob retries a failed job
func (h *JobHandler) RetryJob(c *gin.Context) {
	jobID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	h.logger.Info("Retrying job", zap.String("job_id", jobID))

	// Get the original job
	job, err := h.stores.Jobs.GetByID(jobID)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		h.logger.Error("Failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
		})
		return
	}

	if job.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if job can be retried
	if job.Status != "failed" && job.Status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot retry job in %s state", job.Status),
		})
		return
	}

	// Create new job ID for the retry
	newJobID := uuid.New().String()

	h.logger.Info("Creating retry job",
		zap.String("original_job_id", jobID),
		zap.String("new_job_id", newJobID),
	)

	// Create new job record
	newJob := &store.Job{
		ID:              newJobID,
		UserID:          job.UserID,
		Status:          "analyzing",
		Code:            job.Code,
		Language:        job.Language,
		Qubits:          job.Qubits,
		Depth:           job.Depth,
		GateCount:       job.GateCount,
		ComplexityClass: job.ComplexityClass,
		Method:          job.Method,
		Priority:        job.Priority,
		RetryCount:      job.RetryCount + 1,
	}

	if err := h.stores.Jobs.Create(newJob); err != nil {
		h.logger.Error("Failed to create retry job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create retry job",
		})
		return
	}

	// Create QuantumJob CR for retry
	ctx := context.Background()
	quantumJob := &k8s.QuantumJob{
		ID:       newJobID,
		UserID:   job.UserID,
		Code:     job.Code,
		Language: job.Language,
		Complexity: map[string]interface{}{
			"qubits":     *job.Qubits,
			"depth":      *job.Depth,
			"gateCount":  *job.GateCount,
			"class":      *job.ComplexityClass,
			"method":     *job.Method,
		},
		Scheduling: map[string]interface{}{
			"priority": job.Priority,
			"timeout":  300,
		},
		Resources: map[string]string{
			"cpu":    "4",
			"memory": "4Gi",
			"gpu":    "0",
		},
	}

	err = h.k8sClient.CreateQuantumJob(ctx, quantumJob)
	if err != nil {
		h.logger.Error("Failed to create retry QuantumJob CR", zap.Error(err))
		h.stores.Jobs.UpdateStatus(newJobID, userID.(string), "failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to submit retry job to cluster",
		})
		return
	}

	// Update status
	err = h.stores.Jobs.UpdateStatus(newJobID, userID.(string), "submitted")
	if err != nil {
		h.logger.Warn("Failed to update retry job status", zap.Error(err))
	}

	c.JSON(http.StatusCreated, gin.H{
		"original_job_id": jobID,
		"new_job_id":      newJobID,
		"status":          "submitted",
		"message":         "Job retry submitted successfully",
	})
}

// GetJobResult retrieves the execution result of a completed job
func (h *JobHandler) GetJobResult(c *gin.Context) {
	jobID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get job from database
	job, err := h.stores.Jobs.GetByID(jobID)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		h.logger.Error("Failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
		})
		return
	}

	if job.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if job is completed
	if job.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Job not completed",
			"status": job.Status,
		})
		return
	}

	// Get result from QuantumJob CR or return cached result
	ctx := context.Background()
	quantumJob, err := h.k8sClient.GetQuantumJob(ctx, jobID)
	if err != nil {
		h.logger.Error("Failed to get QuantumJob CR", zap.Error(err))
		// If we have a cached result reference, use that
		if job.ResultRef != nil && *job.ResultRef != "" {
			c.JSON(http.StatusOK, gin.H{
				"job_id":       jobID,
				"status":       job.Status,
				"result_ref":   *job.ResultRef,
				"message":      "Result available in object storage",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve result",
		})
		return
	}

	// Mock result structure (would be actual quantum simulation results)
	result := gin.H{
		"job_id":           jobID,
		"status":           job.Status,
		"execution_time":   job.ExecutionTimeMs,
		"started_at":       job.StartedAt,
		"completed_at":     job.CompletedAt,
		"assigned_node":    job.AssignedNode,
		"assigned_pool":    job.AssignedPool,
		"result": gin.H{
			"counts": gin.H{
				"00": 256,
				"01": 244,
				"10": 255,
				"11": 269,
			},
			"shots": 1024,
			"success": true,
		},
		"metadata": gin.H{
			"qubits":     *job.Qubits,
			"depth":      *job.Depth,
			"gate_count": *job.GateCount,
			"method":     *job.Method,
		},
	}

	if quantumJob.ResultRef != "" {
		result["result_ref"] = quantumJob.ResultRef
	}

	c.JSON(http.StatusOK, result)
}

// GetJobLogs retrieves the execution logs of a job
func (h *JobHandler) GetJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify job ownership
	job, err := h.stores.Jobs.GetByID(jobID)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		h.logger.Error("Failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
		})
		return
	}

	if job.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.logger.Info("Retrieving job logs", zap.String("job_id", jobID))

	// Get logs from Kubernetes pods
	ctx := context.Background()
	logs, err := h.k8sClient.GetPodLogs(ctx, jobID)
	if err != nil {
		h.logger.Error("Failed to get pod logs", zap.Error(err))
		
		// Provide a fallback response based on job status
		var fallbackLogs string
		switch job.Status {
		case "pending", "analyzing":
			fallbackLogs = "Job is pending analysis and scheduling...\n"
		case "submitted", "scheduling":
			fallbackLogs = "Job submitted to cluster, waiting for scheduling...\n"
		case "running":
			fallbackLogs = "Job is currently running on cluster...\n(Pod logs temporarily unavailable)\n"
		case "failed":
			fallbackLogs = fmt.Sprintf("Job failed with error: %s\n", job.ErrorMessage)
		case "completed":
			fallbackLogs = "Job completed successfully.\n(Detailed logs may have been archived)\n"
		case "cancelled":
			fallbackLogs = "Job was cancelled by user.\n"
		default:
			fallbackLogs = fmt.Sprintf("Job status: %s\n", job.Status)
		}

		c.JSON(http.StatusOK, gin.H{
			"job_id":    jobID,
			"status":    job.Status,
			"logs":      fallbackLogs,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"source":    "fallback",
		})
		return
	}

	// Parse query parameters for log filtering
	since := c.Query("since")      // e.g., "1h", "30m"
	tail := c.Query("tail")        // number of lines to tail
	follow := c.Query("follow")    // "true" for streaming

	response := gin.H{
		"job_id":    jobID,
		"status":    job.Status,
		"logs":      logs,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"source":    "kubernetes",
	}

	// Add filtering info if provided
	if since != "" {
		response["since"] = since
	}
	if tail != "" {
		response["tail"] = tail
	}
	if follow == "true" {
		response["follow"] = true
		// TODO: Implement WebSocket streaming for real-time logs
		response["note"] = "Real-time log streaming not yet implemented"
	}

	c.JSON(http.StatusOK, response)
}