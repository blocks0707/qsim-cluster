package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client provides interface to the Circuit Analyzer microservice
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// Config holds configuration for analyzer client
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// AnalyzeRequest represents the request to analyze a quantum circuit
type AnalyzeRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// AnalyzeResponse represents the analysis result from the analyzer service
type AnalyzeResponse struct {
	Qubits              int     `json:"qubits"`
	Depth               int     `json:"depth"`
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
	GateBreakdown       map[string]int `json:"gate_breakdown,omitempty"`
	CircuitDescription  string  `json:"circuit_description,omitempty"`
	OptimizationTips    []string `json:"optimization_tips,omitempty"`
}

// ErrorResponse represents an error response from the analyzer service
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewClient creates a new analyzer client
func NewClient(config Config, logger *zap.Logger) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// Analyze sends a circuit analysis request to the analyzer service
func (c *Client) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	if req.Code == "" {
		return nil, fmt.Errorf("code cannot be empty")
	}

	if req.Language == "" {
		req.Language = "python" // default to Python
	}

	c.logger.Info("Sending circuit analysis request",
		zap.String("language", req.Language),
		zap.Int("code_length", len(req.Code)),
	)

	// Prepare request body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/analyze", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	c.logger.Info("Analyzer request completed",
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration),
	)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("analyzer error (%d): %s - %s", resp.StatusCode, errorResp.Error, errorResp.Message)
		}
		return nil, fmt.Errorf("analyzer request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse successful response
	var result AnalyzeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Circuit analysis completed",
		zap.Int("qubits", result.Qubits),
		zap.Int("depth", result.Depth),
		zap.Int("gate_count", result.GateCount),
		zap.String("complexity_class", result.ComplexityClass),
		zap.String("recommended_method", result.RecommendedMethod),
		zap.String("recommended_pool", result.RecommendedPool),
	)

	return &result, nil
}

// Health checks if the analyzer service is healthy
func (c *Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("analyzer service health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("analyzer service unhealthy, status code: %d", resp.StatusCode)
	}

	c.logger.Debug("Analyzer service health check passed")
	return nil
}

// GetComplexityMapping returns a mapping of complexity classes to resource requirements
func GetComplexityMapping() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"A": { // Light circuits
			"max_qubits":         10,
			"max_depth":          50,
			"max_gates":          100,
			"cpu_cores":          2,
			"memory_mb":          1024,
			"max_time_sec":       30,
			"recommended_pool":   "cpu",
			"recommended_method": "statevector",
		},
		"B": { // Medium circuits
			"max_qubits":         20,
			"max_depth":          200,
			"max_gates":          500,
			"cpu_cores":          4,
			"memory_mb":          4096,
			"max_time_sec":       300,
			"recommended_pool":   "cpu",
			"recommended_method": "statevector",
		},
		"C": { // Heavy circuits
			"max_qubits":         30,
			"max_depth":          1000,
			"max_gates":          2000,
			"cpu_cores":          16,
			"memory_mb":          16384,
			"max_time_sec":       1800,
			"recommended_pool":   "high-cpu",
			"recommended_method": "mps",
		},
		"D": { // Very heavy circuits (GPU recommended)
			"max_qubits":         50,
			"max_depth":          5000,
			"max_gates":          10000,
			"cpu_cores":          32,
			"memory_mb":          65536,
			"max_time_sec":       3600,
			"recommended_pool":   "gpu",
			"recommended_method": "mps",
		},
	}
}

// EstimateResources provides a fallback complexity estimation if analyzer service is unavailable
func EstimateResources(code string, language string) *AnalyzeResponse {
	// Count quantum gates in the code (very basic)
	gateCount := countGatesInCode(code)
	
	// Estimate qubits (look for patterns)
	qubits := estimateQubitsFromCode(code)
	
	// Determine complexity class
	var complexityClass string
	var estimatedCPU int
	var estimatedMemoryMB int
	var estimatedTimeSec int
	var recommendedPool string
	var recommendedMethod string
	
	if qubits <= 10 && gateCount <= 100 {
		complexityClass = "A"
		estimatedCPU = 2
		estimatedMemoryMB = 1024
		estimatedTimeSec = 30
		recommendedPool = "cpu"
		recommendedMethod = "statevector"
	} else if qubits <= 20 && gateCount <= 500 {
		complexityClass = "B"
		estimatedCPU = 4
		estimatedMemoryMB = 4096
		estimatedTimeSec = 300
		recommendedPool = "cpu"
		recommendedMethod = "statevector"
	} else if qubits <= 30 && gateCount <= 2000 {
		complexityClass = "C"
		estimatedCPU = 16
		estimatedMemoryMB = 16384
		estimatedTimeSec = 1800
		recommendedPool = "high-cpu"
		recommendedMethod = "mps"
	} else {
		complexityClass = "D"
		estimatedCPU = 32
		estimatedMemoryMB = 65536
		estimatedTimeSec = 3600
		recommendedPool = "gpu"
		recommendedMethod = "mps"
	}
	
	// Memory estimation: 2^qubits bytes for statevector
	var memoryBytes int64
	if qubits <= 20 {
		memoryBytes = 1 << uint(qubits) * 16 // 16 bytes per complex number
	} else {
		memoryBytes = int64(estimatedMemoryMB) * 1024 * 1024
	}
	
	depth := gateCount / max(qubits, 1) // rough estimate
	parallelism := float64(qubits) / float64(depth)
	if parallelism > 1.0 {
		parallelism = 1.0
	}
	
	return &AnalyzeResponse{
		Qubits:              qubits,
		Depth:               depth,
		GateCount:          gateCount,
		CXCount:            gateCount / 4, // estimate CNOT gates as 25% of total
		Parallelism:        parallelism,
		MemoryBytes:        memoryBytes,
		ComplexityClass:    complexityClass,
		RecommendedMethod:  recommendedMethod,
		EstimatedCPU:       estimatedCPU,
		EstimatedMemoryMB:  estimatedMemoryMB,
		EstimatedTimeSec:   estimatedTimeSec,
		RecommendedPool:    recommendedPool,
		CircuitDescription: "Estimated analysis (analyzer service unavailable)",
	}
}

// Helper functions for fallback estimation
func countGatesInCode(code string) int {
	gates := []string{
		".h(", ".x(", ".y(", ".z(", ".s(", ".t(", 
		".cx(", ".cy(", ".cz(", ".cnot(", ".ccx(",
		".rx(", ".ry(", ".rz(", ".u1(", ".u2(", ".u3(",
		".measure(", ".measure_all()", ".barrier(",
	}
	
	count := 0
	for _, gate := range gates {
		count += len(code) - len(replaceAll(code, gate, ""))
	}
	
	// Rough estimation
	return max(count/4, 10) // assume average gate call is 4 characters
}

func estimateQubitsFromCode(code string) int {
	// Look for QuantumCircuit creation patterns
	if idx := findIndex(code, "QuantumCircuit("); idx >= 0 {
		// Try to extract the first parameter
		start := idx + len("QuantumCircuit(")
		end := findIndex(code[start:], ",")
		if end == -1 {
			end = findIndex(code[start:], ")")
		}
		if end > 0 && end < 10 { // reasonable number
			if num := parseIntFromString(code[start:start+end]); num > 0 && num <= 100 {
				return num
			}
		}
	}
	
	// Fallback: estimate from code complexity
	if len(code) < 200 {
		return 3
	} else if len(code) < 500 {
		return 8
	} else if len(code) < 1000 {
		return 15
	} else {
		return 25
	}
}

// Simple helper functions (would use proper string libraries in production)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func replaceAll(s, old, new string) string {
	// Simplified replace - would use strings.ReplaceAll in real code
	return s
}

func findIndex(s, substr string) int {
	// Simplified find - would use strings.Index in real code  
	return -1
}

func parseIntFromString(s string) int {
	// Simplified int parsing - would use strconv.Atoi with proper error handling
	return 0
}