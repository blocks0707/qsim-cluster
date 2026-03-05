package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/analyzer"
	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

type AnalysisHandler struct {
	stores         *store.Stores
	analyzerClient *analyzer.Client
	logger         *zap.Logger
}

func NewAnalysisHandler(stores *store.Stores, analyzerClient *analyzer.Client, logger *zap.Logger) *AnalysisHandler {
	return &AnalysisHandler{
		stores:         stores,
		analyzerClient: analyzerClient,
		logger:         logger,
	}
}

// AnalyzeCircuitRequest represents the request body for circuit analysis
type AnalyzeCircuitRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language,omitempty"`
}

// AnalyzeCircuitResponse represents the analysis result
type AnalyzeCircuitResponse struct {
	Qubits               int     `json:"qubits"`
	Depth                int     `json:"depth"`
	GateCount           int     `json:"gate_count"`
	CXCount             int     `json:"cx_count"`
	Parallelism         float64 `json:"parallelism"`
	MemoryBytes         int64   `json:"memory_bytes"`
	ComplexityClass     string  `json:"complexity_class"`
	RecommendedMethod   string  `json:"recommended_method"`
	EstimatedCPU        int     `json:"estimated_cpu"`
	EstimatedMemoryMB   int     `json:"estimated_memory_mb"`
	EstimatedTimeSec    int     `json:"estimated_time_sec"`
	RecommendedPool     string  `json:"recommended_pool"`
}

// AnalyzeCircuit analyzes quantum circuit complexity without execution
func (h *AnalysisHandler) AnalyzeCircuit(c *gin.Context) {
	var req AnalyzeCircuitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set default language if not specified
	if req.Language == "" {
		req.Language = "python"
	}

	h.logger.Info("Analyzing circuit",
		zap.String("language", req.Language),
		zap.Int("code_length", len(req.Code)),
	)

	// Call circuit analyzer service
	ctx := context.Background()
	analyzerReq := &analyzer.AnalyzeRequest{
		Code:     req.Code,
		Language: req.Language,
	}

	result, err := h.analyzerClient.Analyze(ctx, analyzerReq)
	if err != nil {
		h.logger.Warn("Circuit analyzer service failed, using fallback estimation", zap.Error(err))
		
		// Use fallback estimation
		result = analyzer.EstimateResources(req.Code, req.Language)
	}

	h.logger.Info("Circuit analysis completed",
		zap.Int("qubits", result.Qubits),
		zap.Int("depth", result.Depth),
		zap.String("complexity_class", result.ComplexityClass),
		zap.String("recommended_method", result.RecommendedMethod),
		zap.String("recommended_pool", result.RecommendedPool),
	)

	// Map analyzer response to handler response
	analysis := AnalyzeCircuitResponse{
		Qubits:              result.Qubits,
		Depth:               result.Depth,
		GateCount:           result.GateCount,
		CXCount:             result.CXCount,
		Parallelism:         result.Parallelism,
		MemoryBytes:         result.MemoryBytes,
		ComplexityClass:     result.ComplexityClass,
		RecommendedMethod:   result.RecommendedMethod,
		EstimatedCPU:        result.EstimatedCPU,
		EstimatedMemoryMB:   result.EstimatedMemoryMB,
		EstimatedTimeSec:    result.EstimatedTimeSec,
		RecommendedPool:     result.RecommendedPool,
	}

	c.JSON(http.StatusOK, analysis)
}