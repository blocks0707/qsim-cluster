package k8s

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes clients and provides quantum-specific operations
type Client struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
	logger        *zap.Logger
}

// Config holds configuration for K8s client
type Config struct {
	KubeConfig string
	InCluster  bool
}

// QuantumJob represents the QuantumJob custom resource
type QuantumJob struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Status          string                 `json:"status"`
	Code            string                 `json:"code"`
	Language        string                 `json:"language"`
	Complexity      map[string]interface{} `json:"complexity"`
	Scheduling      map[string]interface{} `json:"scheduling"`
	Resources       map[string]string      `json:"resources"`
	AssignedNode    string                 `json:"assigned_node,omitempty"`
	AssignedPool    string                 `json:"assigned_pool,omitempty"`
	StartTime       *time.Time             `json:"start_time,omitempty"`
	CompletionTime  *time.Time             `json:"completion_time,omitempty"`
	ExecutionTimeMs *int64                 `json:"execution_time_ms,omitempty"`
	ResultRef       string                 `json:"result_ref,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// NodeInfo represents node information
type NodeInfo struct {
	Name         string            `json:"name"`
	Pool         string            `json:"pool"`
	Status       string            `json:"status"`
	CPUCores     int64             `json:"cpu_cores"`
	MemoryGB     int64             `json:"memory_gb"`
	GPU          bool              `json:"gpu"`
	GPUType      string            `json:"gpu_type,omitempty"`
	ActiveJobs   int               `json:"active_jobs"`
	CPUUsage     string            `json:"cpu_usage"`
	MemoryUsage  string            `json:"memory_usage"`
	GPUUsage     string            `json:"gpu_usage,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// ClusterStatus represents overall cluster status
type ClusterStatus struct {
	Status        string            `json:"status"`
	Version       string            `json:"version"`
	TotalNodes    int               `json:"total_nodes"`
	ReadyNodes    int               `json:"ready_nodes"`
	NodePools     map[string]int    `json:"node_pools"`
	JobStats      map[string]int    `json:"job_stats"`
	ResourceUsage map[string]string `json:"resource_usage"`
}

// NewClient creates a new Kubernetes client
func NewClient(config Config, logger *zap.Logger) (*Client, error) {
	var restConfig *rest.Config
	var err error

	if config.InCluster {
		// Use in-cluster configuration
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		kubeconfig := config.KubeConfig
		if kubeconfig == "" {
			kubeconfig = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		}

		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create dynamic client for CRDs
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	client := &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		logger:        logger,
	}

	logger.Info("Kubernetes client initialized successfully")
	return client, nil
}

// DynamicClient returns the underlying dynamic client
func (c *Client) DynamicClient() dynamic.Interface {
	return c.dynamicClient
}

// CreateQuantumJob creates a QuantumJob custom resource
func (c *Client) CreateQuantumJob(ctx context.Context, job *QuantumJob) error {
	// QuantumJob CRD schema
	gvr := schema.GroupVersionResource{
		Group:    "quantum.blocksq.io",
		Version:  "v1alpha1",
		Resource: "quantumjobs",
	}

	// Build the custom resource object
	quantumJob := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "quantum.blocksq.io/v1alpha1",
			"kind":       "QuantumJob",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("qjob-%s", job.ID),
				"namespace": "quantum-jobs",
				"labels": map[string]interface{}{
					"user":             job.UserID,
					"complexity-class": getComplexityClass(job.Complexity),
					"quantum-job-id":   job.ID,
				},
			},
			"spec": map[string]interface{}{
				"userID":  job.UserID,
				"circuit": map[string]interface{}{
					"source":   job.Code,
					"language": job.Language,
				},
				"complexity":  job.Complexity,
				"scheduling":  job.Scheduling,
				"resources":   job.Resources,
			},
			"status": map[string]interface{}{
				"phase": "Pending",
				"conditions": []map[string]interface{}{
					{
						"type":               "Submitted",
						"status":             "True",
						"lastTransitionTime": time.Now().UTC().Format(time.RFC3339),
						"message":            "Job submitted to cluster",
					},
				},
			},
		},
	}

	_, err := c.dynamicClient.Resource(gvr).Namespace("quantum-jobs").Create(ctx, quantumJob, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create QuantumJob: %w", err)
	}

	c.logger.Info("QuantumJob created successfully", zap.String("job_id", job.ID))
	return nil
}

// GetQuantumJob retrieves a QuantumJob custom resource
func (c *Client) GetQuantumJob(ctx context.Context, jobID string) (*QuantumJob, error) {
	gvr := schema.GroupVersionResource{
		Group:    "quantum.blocksq.io",
		Version:  "v1alpha1",
		Resource: "quantumjobs",
	}

	obj, err := c.dynamicClient.Resource(gvr).Namespace("quantum-jobs").Get(ctx, fmt.Sprintf("qjob-%s", jobID), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get QuantumJob: %w", err)
	}

	// Parse the unstructured object back to QuantumJob
	job := &QuantumJob{
		ID: jobID,
	}

	// Extract spec
	if spec, found, err := unstructured.NestedMap(obj.Object, "spec"); err == nil && found {
		if circuit, found, _ := unstructured.NestedMap(spec, "circuit"); found {
			if code, found, _ := unstructured.NestedString(circuit, "source"); found {
				job.Code = code
			}
			if lang, found, _ := unstructured.NestedString(circuit, "language"); found {
				job.Language = lang
			}
		}
		if complexity, found, _ := unstructured.NestedMap(spec, "complexity"); found {
			job.Complexity = complexity
		}
		if resources, found, _ := unstructured.NestedMap(spec, "resources"); found {
			job.Resources = make(map[string]string)
			for k, v := range resources {
				if str, ok := v.(string); ok {
					job.Resources[k] = str
				}
			}
		}
	}

	// Extract status
	if status, found, err := unstructured.NestedMap(obj.Object, "status"); err == nil && found {
		if phase, found, _ := unstructured.NestedString(status, "phase"); found {
			job.Status = phase
		}
		if node, found, _ := unstructured.NestedString(status, "assignedNode"); found {
			job.AssignedNode = node
		}
		if pool, found, _ := unstructured.NestedString(status, "assignedPool"); found {
			job.AssignedPool = pool
		}
		if resultRef, found, _ := unstructured.NestedString(status, "resultRef", "key"); found {
			job.ResultRef = resultRef
		}
	}

	return job, nil
}

// DeleteQuantumJob deletes a QuantumJob custom resource
func (c *Client) DeleteQuantumJob(ctx context.Context, jobID string) error {
	gvr := schema.GroupVersionResource{
		Group:    "quantum.blocksq.io",
		Version:  "v1alpha1",
		Resource: "quantumjobs",
	}

	err := c.dynamicClient.Resource(gvr).Namespace("quantum-jobs").Delete(ctx, fmt.Sprintf("qjob-%s", jobID), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete QuantumJob: %w", err)
	}

	c.logger.Info("QuantumJob deleted successfully", zap.String("job_id", jobID))
	return nil
}

// GetPodLogs retrieves logs from a job's pod
func (c *Client) GetPodLogs(ctx context.Context, jobID string) (string, error) {
	// Find the pod for this job
	pods, err := c.clientset.CoreV1().Pods("quantum-jobs").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("quantum-job=%s", jobID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", jobID)
	}

	// Get logs from the first pod (should only be one)
	pod := pods.Items[0]
	req := c.clientset.CoreV1().Pods("quantum-jobs").GetLogs(pod.Name, &corev1.PodLogOptions{})
	logs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer logs.Close()

	// Read logs (simplified - in production would stream properly)
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, _ := logs.Read(buf)
	return string(buf[:n]), nil
}

// GetClusterStatus returns overall cluster status
func (c *Client) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	// Get nodes
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Count node pools and ready nodes
	readyNodes := 0
	nodePools := make(map[string]int)

	for _, node := range nodes.Items {
		// Check if node is ready
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				readyNodes++
				break
			}
		}

		// Count pools based on labels
		if pool, exists := node.Labels["node-pool"]; exists {
			nodePools[pool]++
		} else {
			nodePools["default"]++
		}
	}

	// Get job statistics (simplified - would query QuantumJobs CRD)
	jobStats := map[string]int{
		"total":     0,
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}

	status := &ClusterStatus{
		Status:     "healthy",
		Version:    "v0.1.0",
		TotalNodes: len(nodes.Items),
		ReadyNodes: readyNodes,
		NodePools:  nodePools,
		JobStats:   jobStats,
		ResourceUsage: map[string]string{
			"cpu_usage":    "25%",
			"memory_usage": "40%",
			"gpu_usage":    "0%",
		},
	}

	return status, nil
}

// ListNodes returns information about cluster nodes
func (c *Client) ListNodes(ctx context.Context) ([]*NodeInfo, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var nodeInfos []*NodeInfo
	for _, node := range nodes.Items {
		info := &NodeInfo{
			Name:   node.Name,
			Pool:   getNodePool(node.Labels),
			Status: getNodeStatus(&node),
			Labels: node.Labels,
		}

		// Extract resource information
		if cpu := node.Status.Capacity[corev1.ResourceCPU]; !cpu.IsZero() {
			info.CPUCores = cpu.Value()
		}
		if memory := node.Status.Capacity[corev1.ResourceMemory]; !memory.IsZero() {
			info.MemoryGB = memory.Value() / (1024 * 1024 * 1024) // Convert bytes to GB
		}

		// Check for GPU
		if gpu, exists := node.Status.Capacity["nvidia.com/gpu"]; exists && !gpu.IsZero() {
			info.GPU = true
			if gpuType, exists := node.Labels["gpu.nvidia.com/class"]; exists {
				info.GPUType = gpuType
			}
		}

		// Get metrics (simplified - in production would query metrics server)
		info.CPUUsage = "20%"
		info.MemoryUsage = "35%"
		info.ActiveJobs = 0

		nodeInfos = append(nodeInfos, info)
	}

	return nodeInfos, nil
}

// Helper functions
func getComplexityClass(complexity map[string]interface{}) string {
	if class, exists := complexity["class"]; exists {
		if str, ok := class.(string); ok {
			return str
		}
	}
	return "B" // default
}

func getNodePool(labels map[string]string) string {
	if pool, exists := labels["node-pool"]; exists {
		return pool
	}
	if pool, exists := labels["quantum.blocksq.io/pool"]; exists {
		return pool
	}
	return "default"
}

func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "ready"
			}
			return "not-ready"
		}
	}
	return "unknown"
}