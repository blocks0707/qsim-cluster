/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

// QuantumNodeProfileReconciler reconciles a QuantumNodeProfile object
type QuantumNodeProfileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumnodeprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumnodeprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumnodeprofiles/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=metrics.k8s.io,resources=nodes,verbs=get;list
//+kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=get;list

// Reconcile handles QuantumNodeProfile reconciliation
func (r *QuantumNodeProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the QuantumNodeProfile instance
	var nodeProfile quantumv1alpha1.QuantumNodeProfile
	if err := r.Get(ctx, req.NamespacedName, &nodeProfile); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("QuantumNodeProfile not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get QuantumNodeProfile")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling QuantumNodeProfile", "name", nodeProfile.Name)

	// Update node profile status with current load and availability
	if err := r.updateNodeProfileStatus(ctx, &nodeProfile); err != nil {
		logger.Error(err, "Failed to update QuantumNodeProfile status")
		return ctrl.Result{}, err
	}

	// Reconcile every 30 seconds for load monitoring
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// updateNodeProfileStatus updates the status section with current node metrics
func (r *QuantumNodeProfileReconciler) updateNodeProfileStatus(ctx context.Context, nodeProfile *quantumv1alpha1.QuantumNodeProfile) error {
	logger := log.FromContext(ctx)

	// Get the corresponding Kubernetes node
	nodeName := nodeProfile.Name
	var node corev1.Node
	if err := r.Get(ctx, types.NamespacedName{Name: nodeName}, &node); err != nil {
		if errors.IsNotFound(err) {
			// Node doesn't exist, mark profile as not ready
			return r.updateNodeProfileNotReady(ctx, nodeProfile, "NodeNotFound", "Kubernetes node not found")
		}
		return err
	}

	// Check if node is ready
	nodeReady := r.isNodeReady(&node)
	if !nodeReady {
		return r.updateNodeProfileNotReady(ctx, nodeProfile, "NodeNotReady", "Kubernetes node is not ready")
	}

	// Get current resource usage
	cpuUsage, memoryUsage, err := r.getNodeResourceUsage(ctx, nodeName)
	if err != nil {
		logger.Error(err, "Failed to get node resource usage, using default values")
		cpuUsage = 0.0
		memoryUsage = 0.0
	}

	// Count active quantum jobs on this node
	activeJobs, err := r.countActiveQuantumJobs(ctx, nodeName)
	if err != nil {
		logger.Error(err, "Failed to count active quantum jobs")
		activeJobs = 0
	}

	// Update status
	now := metav1.NewTime(time.Now())
	nodeProfile.Status.CurrentLoad = &quantumv1alpha1.LoadStatus{
		CPUUsagePercent:    cpuUsage,
		MemoryUsagePercent: memoryUsage,
		ActiveJobs:         activeJobs,
	}
	nodeProfile.Status.LastUpdated = &now
	nodeProfile.Status.Ready = true

	// Update conditions
	readyCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "NodeReady",
		Message:            "Node is ready and accepting quantum jobs",
	}
	r.setCondition(&nodeProfile.Status.Conditions, readyCondition)

	// Update the status
	if err := r.Status().Update(ctx, nodeProfile); err != nil {
		return fmt.Errorf("failed to update QuantumNodeProfile status: %w", err)
	}

	logger.Info("Updated QuantumNodeProfile status",
		"name", nodeProfile.Name,
		"cpuUsage", cpuUsage,
		"memoryUsage", memoryUsage,
		"activeJobs", activeJobs)

	return nil
}

// updateNodeProfileNotReady marks the node profile as not ready
func (r *QuantumNodeProfileReconciler) updateNodeProfileNotReady(ctx context.Context, nodeProfile *quantumv1alpha1.QuantumNodeProfile, reason, message string) error {
	now := metav1.NewTime(time.Now())
	nodeProfile.Status.Ready = false
	nodeProfile.Status.LastUpdated = &now

	notReadyCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
	r.setCondition(&nodeProfile.Status.Conditions, notReadyCondition)

	return r.Status().Update(ctx, nodeProfile)
}

// isNodeReady checks if the Kubernetes node is ready
func (r *QuantumNodeProfileReconciler) isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// getNodeResourceUsage retrieves current CPU and memory usage for a node
func (r *QuantumNodeProfileReconciler) getNodeResourceUsage(ctx context.Context, nodeName string) (float64, float64, error) {
	// This is a simplified implementation. In production, you would:
	// 1. Query metrics-server API for node metrics
	// 2. Or integrate with Prometheus for more detailed metrics
	// 3. Or use the metrics client directly

	// For now, return mock values
	// TODO: Implement actual metrics collection
	return 0.0, 0.0, nil
}

// countActiveQuantumJobs counts the number of running quantum simulation pods on a node
func (r *QuantumNodeProfileReconciler) countActiveQuantumJobs(ctx context.Context, nodeName string) (int32, error) {
	var podList corev1.PodList

	// List all pods running on the specified node
	if err := r.List(ctx, &podList, client.MatchingFields{"spec.nodeName": nodeName}); err != nil {
		return 0, fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	var activeJobs int32
	for _, pod := range podList.Items {
		// Check if this is a quantum job pod and is running
		if r.isQuantumJobPod(&pod) && pod.Status.Phase == corev1.PodRunning {
			activeJobs++
		}
	}

	return activeJobs, nil
}

// isQuantumJobPod checks if a pod belongs to a quantum job
func (r *QuantumNodeProfileReconciler) isQuantumJobPod(pod *corev1.Pod) bool {
	// Check for quantum job labels
	if jobName, exists := pod.Labels["quantum-job"]; exists && jobName != "" {
		return true
	}
	return false
}

// setCondition sets or updates a condition in the conditions slice
func (r *QuantumNodeProfileReconciler) setCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	for i, condition := range *conditions {
		if condition.Type == newCondition.Type {
			(*conditions)[i] = newCondition
			return
		}
	}
	// Condition not found, append it
	*conditions = append(*conditions, newCondition)
}

// SetupWithManager sets up the controller with the Manager
func (r *QuantumNodeProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index pods by node name for efficient queries
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "spec.nodeName", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		if pod.Spec.NodeName == "" {
			return nil
		}
		return []string{pod.Spec.NodeName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&quantumv1alpha1.QuantumNodeProfile{}).
		Complete(r)
}

// TODO: Implement event handlers for node and pod changes
// This would trigger reconciliation when nodes join/leave or when pods start/stop