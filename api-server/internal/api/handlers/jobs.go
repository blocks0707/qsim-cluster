package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

type JobHandler struct {
	stores *store.Stores
	logger *zap.Logger
}

func NewJobHandler(stores *store.Stores, logger *zap.Logger) *JobHandler {
	return &JobHandler{
		stores: stores,
		logger: logger,
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

	h.logger.Info("Creating quantum job",
		zap.String("job_id", jobID),
		zap.String("user_id", userID.(string)),
		zap.String("language", req.Language),
	)

	// TODO: Analyze circuit complexity
	// TODO: Create QuantumJob CR in Kubernetes
	// For now, just store in database as pending

	job := &store.Job{
		ID:       jobID,
		UserID:   userID.(string),
		Status:   "pending",
		Code:     req.Code,
		Language: req.Language,
	}

	if err := h.stores.Jobs.Create(job); err != nil {
		h.logger.Error("Failed to create job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create job",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"job_id": jobID,
		"status": "pending",
		"message": "Job created successfully",
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

	// TODO: Cancel the QuantumJob CR in Kubernetes
	// For now, just update status in database

	err := h.stores.Jobs.UpdateStatus(jobID, userID.(string), "cancelled")
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		h.logger.Error("Failed to cancel job", zap.Error(err))
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

	// TODO: Implement job retry logic
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Retry not implemented yet",
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

	// TODO: Get result from object storage
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Result retrieval not implemented yet",
	})
}

// GetJobLogs retrieves the execution logs of a job
func (h *JobHandler) GetJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// TODO: Get logs from Kubernetes pods
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Log retrieval not implemented yet",
	})
}