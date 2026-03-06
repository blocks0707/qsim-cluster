package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/mungch0120/qsim-cluster/api-server/internal/k8s"
)

type JupyterHandler struct {
	k8sClient *k8s.Client
	logger    *zap.Logger
}

func NewJupyterHandler(k8sClient *k8s.Client, logger *zap.Logger) *JupyterHandler {
	return &JupyterHandler{k8sClient: k8sClient, logger: logger}
}

var jupyterGVR = schema.GroupVersionResource{
	Group:    "quantum.blocksq.io",
	Version:  "v1alpha1",
	Resource: "jupyterruntimes",
}

type CreateJupyterRequest struct {
	Image    string   `json:"image,omitempty"`
	CPU      string   `json:"cpu,omitempty"`
	Memory   string   `json:"memory,omitempty"`
	Storage  string   `json:"storage,omitempty"`
	Timeout  int32    `json:"timeout,omitempty"`
	Packages []string `json:"packages,omitempty"`
}

// CreateJupyter creates a new Jupyter notebook session
func (h *JupyterHandler) CreateJupyter(c *gin.Context) {
	var req CreateJupyterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body with defaults
		req = CreateJupyterRequest{}
	}

	userID, _ := c.Get("user_id")

	// Generate a unique name
	name := fmt.Sprintf("%s-%d", userID, metav1.Now().Unix())

	image := req.Image
	if image == "" {
		image = "jupyter/scipy-notebook:latest"
	}
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 3600
	}

	// Build packages list
	packages := req.Packages
	// Always include qiskit
	hasQiskit := false
	for _, p := range packages {
		if p == "qiskit" || p == "qiskit-aer" {
			hasQiskit = true
		}
	}
	if !hasQiskit {
		packages = append([]string{"qiskit", "qiskit-aer"}, packages...)
	}

	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "quantum.blocksq.io/v1alpha1",
			"kind":       "JupyterRuntime",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "quantum-jobs",
			},
			"spec": map[string]interface{}{
				"userID":  userID,
				"image":   image,
				"timeout": timeout,
				"packages": func() []interface{} {
					result := make([]interface{}, len(packages))
					for i, p := range packages {
						result[i] = p
					}
					return result
				}(),
				"resources": map[string]interface{}{
					"cpu":     orDefault(req.CPU, "2"),
					"memory":  orDefault(req.Memory, "4Gi"),
					"storage": orDefault(req.Storage, "10Gi"),
				},
				"qsimEndpoint": "http://api-server.quantum-system.svc:8080",
			},
		},
	}

	dynClient := h.k8sClient.DynamicClient()
	_, err := dynClient.Resource(jupyterGVR).Namespace("quantum-jobs").Create(
		context.Background(), cr, metav1.CreateOptions{})
	if err != nil {
		h.logger.Error("Failed to create JupyterRuntime", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notebook session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":    name,
		"status":  "pending",
		"message": "Jupyter notebook session created",
	})
}

// ListJupyter lists active Jupyter sessions for the user
func (h *JupyterHandler) ListJupyter(c *gin.Context) {
	userID, _ := c.Get("user_id")

	dynClient := h.k8sClient.DynamicClient()
	list, err := dynClient.Resource(jupyterGVR).Namespace("quantum-jobs").List(
		context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("quantum.blocksq.io/user=%s", userID),
		})
	if err != nil {
		// Try without label selector
		list, err = dynClient.Resource(jupyterGVR).Namespace("quantum-jobs").List(
			context.Background(), metav1.ListOptions{})
		if err != nil {
			h.logger.Error("Failed to list JupyterRuntimes", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
			return
		}
	}

	var sessions []gin.H
	for _, item := range list.Items {
		spec, _, _ := unstructured.NestedMap(item.Object, "spec")
		status, _, _ := unstructured.NestedMap(item.Object, "status")

		session := gin.H{
			"name":      item.GetName(),
			"created_at": item.GetCreationTimestamp().Format("2006-01-02T15:04:05Z"),
		}

		if spec != nil {
			if uid, ok := spec["userID"]; ok {
				session["user_id"] = uid
			}
		}
		if status != nil {
			if phase, ok := status["phase"]; ok {
				session["phase"] = phase
			}
			if url, ok := status["url"]; ok {
				session["url"] = url
			}
			if token, ok := status["token"]; ok {
				session["token"] = token
			}
		}
		sessions = append(sessions, session)
	}

	if sessions == nil {
		sessions = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// GetJupyter gets a specific Jupyter session
func (h *JupyterHandler) GetJupyter(c *gin.Context) {
	name := c.Param("name")

	dynClient := h.k8sClient.DynamicClient()
	obj, err := dynClient.Resource(jupyterGVR).Namespace("quantum-jobs").Get(
		context.Background(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	status, _, _ := unstructured.NestedMap(obj.Object, "status")

	c.JSON(http.StatusOK, gin.H{
		"name":       name,
		"spec":       spec,
		"status":     status,
		"created_at": obj.GetCreationTimestamp().Format("2006-01-02T15:04:05Z"),
	})
}

// DeleteJupyter stops and deletes a Jupyter session
func (h *JupyterHandler) DeleteJupyter(c *gin.Context) {
	name := c.Param("name")

	dynClient := h.k8sClient.DynamicClient()
	err := dynClient.Resource(jupyterGVR).Namespace("quantum-jobs").Delete(
		context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted"})
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
