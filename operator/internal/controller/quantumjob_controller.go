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
	quantumruntime "github.com/mungch0120/qsim-cluster/operator/internal/runtime"
	"github.com/mungch0120/qsim-cluster/operator/internal/scheduler"
)

// QuantumJobReconciler reconciles a QuantumJob object
type QuantumJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// Internal components
	PodBuilder *quantumruntime.PodBuilder
	NodeScorer *scheduler.NodeScorer
	Predicates *scheduler.PredicateRegistry
}

//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=quantumnodeprofiles,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile handles QuantumJob lifecycle management
func (r *QuantumJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the QuantumJob instance
	var job quantumv1alpha1.QuantumJob
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("QuantumJob not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get QuantumJob")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling QuantumJob", "name", job.Name, "phase", job.Status.Phase)

	// Handle job based on current phase
	switch job.Status.Phase {
	case "": // New job
		return r.handleNewJob(ctx, &job)
	case quantumv1alpha1.JobPhasePending:
		return r.handlePendingJob(ctx, &job)
	case quantumv1alpha1.JobPhaseAnalyzing:
		return r.handleAnalyzingJob(ctx, &job)
	case quantumv1alpha1.JobPhaseScheduling:
		return r.handleSchedulingJob(ctx, &job)
	case quantumv1alpha1.JobPhaseRunning:
		return r.handleRunningJob(ctx, &job)
	case quantumv1alpha1.JobPhaseSucceeded, quantumv1alpha1.JobPhaseFailed:
		return r.handleCompletedJob(ctx, &job)
	case quantumv1alpha1.JobPhaseCancelled:
		return r.handleCancelledJob(ctx, &job)
	default:
		logger.Info("Unknown job phase", "phase", job.Status.Phase)
		return ctrl.Result{}, nil
	}
}

// handleNewJob initializes a new job and moves it to Pending phase
func (r *QuantumJobReconciler) handleNewJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Handling new QuantumJob", "name", job.Name)

	// Validate the job specification
	if err := r.validateJobSpec(job); err != nil {
		return r.failJob(ctx, job, fmt.Sprintf("Invalid job specification: %v", err))
	}

	// Set initial status
	job.Status.Phase = quantumv1alpha1.JobPhasePending
	r.addJobEvent(job, "Normal", "Created", "QuantumJob created and validation passed")

	// Update status
	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job status to Pending")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// Immediately requeue to process pending state
	return ctrl.Result{Requeue: true}, nil
}

// handlePendingJob moves job to analyzing phase if complexity not provided
func (r *QuantumJobReconciler) handlePendingJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if complexity analysis is already done
	if job.Spec.Complexity != nil {
		// Skip analysis, go directly to scheduling
		job.Status.Phase = quantumv1alpha1.JobPhaseScheduling
		r.addJobEvent(job, "Normal", "SkippedAnalysis", "Circuit complexity already provided")
	} else {
		// Need to perform analysis
		job.Status.Phase = quantumv1alpha1.JobPhaseAnalyzing
		r.addJobEvent(job, "Normal", "StartingAnalysis", "Starting circuit complexity analysis")
	}

	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job phase")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{Requeue: true}, nil
}

// handleAnalyzingJob performs circuit analysis or waits for external analysis
func (r *QuantumJobReconciler) handleAnalyzingJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// In a real implementation, this would:
	// 1. Call the circuit analyzer service
	// 2. Wait for analysis results
	// 3. Update the job spec with complexity metadata

	// For now, provide default complexity if not present
	if job.Spec.Complexity == nil {
		logger.Info("No complexity provided, using defaults", "name", job.Name)

		// Set default complexity values
		job.Spec.Complexity = &quantumv1alpha1.ComplexitySpec{
			Qubits:            4,
			Depth:             10,
			GateCount:         20,
			Parallelism:       0.5,
			EstimatedMemoryMB: 1024,
			EstimatedCPUCores: 2,
			EstimatedTimeSec:  30,
			Method:            quantumv1alpha1.SimulationMethodStatevector,
		}

		// Update default resources based on complexity
		if job.Spec.Resources.CPU == "" {
			job.Spec.Resources.CPU = "2"
		}
		if job.Spec.Resources.Memory == "" {
			job.Spec.Resources.Memory = "4Gi"
		}
	}

	// Save complexity info for status update message
	qubits := job.Spec.Complexity.Qubits
	depth := job.Spec.Complexity.Depth
	method := job.Spec.Complexity.Method

	// Step 1: Update spec (complexity + resources)
	if err := r.Update(ctx, job); err != nil {
		if errors.IsConflict(err) {
			logger.Info("Conflict updating job spec, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to update job with complexity")
		return ctrl.Result{}, err
	}

	// Step 2: Re-fetch to get latest resourceVersion after spec update
	if err := r.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, job); err != nil {
		logger.Error(err, "Failed to re-fetch job after spec update")
		return ctrl.Result{}, err
	}

	// Step 3: Now apply status changes on the fresh object
	job.Status.Phase = quantumv1alpha1.JobPhaseScheduling
	r.addJobEvent(job, "Normal", "AnalysisCompleted",
		fmt.Sprintf("Circuit analysis complete: %d qubits, depth %d, method %s",
			qubits, depth, method))

	now := metav1.NewTime(time.Now())
	condition := metav1.Condition{
		Type:               "Analyzed",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "AnalysisCompleted",
		Message:            "Circuit complexity analysis completed successfully",
	}
	r.setJobCondition(&job.Status.Conditions, condition)

	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job status")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{Requeue: true}, nil
}

// handleSchedulingJob performs node selection and scheduling
func (r *QuantumJobReconciler) handleSchedulingJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get all available node profiles
	var nodeProfiles quantumv1alpha1.QuantumNodeProfileList
	if err := r.List(ctx, &nodeProfiles); err != nil {
		logger.Error(err, "Failed to list node profiles")
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	if len(nodeProfiles.Items) == 0 {
		logger.Info("No node profiles available, waiting...")
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Convert to slice of pointers for predicates
	var nodes []*quantumv1alpha1.QuantumNodeProfile
	for i := range nodeProfiles.Items {
		nodes = append(nodes, &nodeProfiles.Items[i])
	}

	// Apply predicates to filter suitable nodes
	filteredNodes, filterResults := r.Predicates.Filter(job, nodes)

	if len(filteredNodes) == 0 {
		// No suitable nodes found
		reasons := "No suitable nodes found:"
		for _, result := range filterResults {
			if !result.Passed {
				reasons += fmt.Sprintf(" %s(%s)", result.NodeName, result.Reason)
			}
		}

		logger.Info("No suitable nodes for job", "reasons", reasons)
		r.addJobEvent(job, "Warning", "NoSuitableNodes", reasons)

		// Requeue after a delay
		return ctrl.Result{RequeueAfter: time.Second * 60}, nil
	}

	// Score the filtered nodes
	bestNode, bestScore, err := r.NodeScorer.GetBestNode(job, filteredNodes)
	if err != nil {
		logger.Error(err, "Failed to score nodes")
		return ctrl.Result{}, err
	}

	// Assign the job to the best node
	job.Status.AssignedNode = bestNode.Name
	job.Status.AssignedPool = bestNode.Spec.Pool
	job.Status.Phase = quantumv1alpha1.JobPhaseRunning

	// Set scheduled condition
	now := metav1.NewTime(time.Now())
	condition := metav1.Condition{
		Type:               "Scheduled",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "NodeAssigned",
		Message:            fmt.Sprintf("Assigned to node %s in pool %s (score: %.3f)", bestNode.Name, bestNode.Spec.Pool, bestScore.TotalScore),
	}
	r.setJobCondition(&job.Status.Conditions, condition)

	r.addJobEvent(job, "Normal", "Scheduled",
		fmt.Sprintf("Scheduled to node %s (pool: %s, score: %.3f)", bestNode.Name, bestNode.Spec.Pool, bestScore.TotalScore))

	logger.Info("Job scheduled successfully",
		"job", job.Name,
		"node", bestNode.Name,
		"pool", bestNode.Spec.Pool,
		"score", bestScore.TotalScore)

	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job status after scheduling")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{Requeue: true}, nil
}

// handleRunningJob manages job execution
func (r *QuantumJobReconciler) handleRunningJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if pod already exists
	podName := fmt.Sprintf("qjob-%s-runner", job.Name)
	var existingPod corev1.Pod
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: job.Namespace}, &existingPod)

	if errors.IsNotFound(err) {
		// Pod doesn't exist, create it
		return r.createSimulationPod(ctx, job)
	} else if err != nil {
		logger.Error(err, "Failed to get simulation pod")
		return ctrl.Result{}, err
	}

	// Pod exists, check its status
	return r.monitorSimulationPod(ctx, job, &existingPod)
}

// createSimulationPod creates the simulation pod and configmap
func (r *QuantumJobReconciler) createSimulationPod(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create ConfigMap with user code
	configMap := r.PodBuilder.BuildCodeConfigMap(job)
	if err := r.Create(ctx, configMap); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create ConfigMap")
		return ctrl.Result{}, err
	}

	// Create simulation pod
	pod, err := r.PodBuilder.BuildSimulationPod(job)
	if err != nil {
		return r.failJob(ctx, job, fmt.Sprintf("Failed to build simulation pod: %v", err))
	}

	// Set node affinity if pool is specified
	if job.Spec.Scheduling.NodePool != quantumv1alpha1.NodePoolAuto {
		r.PodBuilder.SetPodNodeAffinity(pod, job.Spec.Scheduling.NodePool)
	}

	if err := r.Create(ctx, pod); err != nil {
		logger.Error(err, "Failed to create simulation pod")
		return r.failJob(ctx, job, fmt.Sprintf("Failed to create simulation pod: %v", err))
	}

	// Set start time
	now := metav1.NewTime(time.Now())
	job.Status.StartTime = &now

	r.addJobEvent(job, "Normal", "PodCreated", fmt.Sprintf("Simulation pod %s created", pod.Name))

	logger.Info("Simulation pod created", "job", job.Name, "pod", pod.Name)

	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job status after pod creation")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// Requeue to monitor pod status
	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

// monitorSimulationPod monitors the running simulation pod
func (r *QuantumJobReconciler) monitorSimulationPod(ctx context.Context, job *quantumv1alpha1.QuantumJob, pod *corev1.Pod) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	switch pod.Status.Phase {
	case corev1.PodPending:
		logger.Info("Simulation pod is pending", "pod", pod.Name)
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil

	case corev1.PodRunning:
		logger.Info("Simulation pod is running", "pod", pod.Name)
		return ctrl.Result{RequeueAfter: time.Second * 15}, nil

	case corev1.PodSucceeded:
		return r.handleJobSuccess(ctx, job, pod)

	case corev1.PodFailed:
		return r.handleJobFailure(ctx, job, pod)

	default:
		logger.Info("Unknown pod phase", "pod", pod.Name, "phase", pod.Status.Phase)
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
}

// handleJobSuccess processes successful job completion
func (r *QuantumJobReconciler) handleJobSuccess(ctx context.Context, job *quantumv1alpha1.QuantumJob, pod *corev1.Pod) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	now := metav1.NewTime(time.Now())
	job.Status.Phase = quantumv1alpha1.JobPhaseSucceeded
	job.Status.CompletionTime = &now

	// Calculate execution time
	if job.Status.StartTime != nil {
		executionTime := int32(now.Sub(job.Status.StartTime.Time).Seconds())
		job.Status.ExecutionTimeSec = &executionTime
	}

	// Set result reference (mock for now)
	job.Status.ResultRef = &quantumv1alpha1.ResultRef{
		Bucket: "quantum-results",
		Key:    fmt.Sprintf("qjob-%s/result.json", job.Name),
	}

	// Set completed condition
	condition := metav1.Condition{
		Type:               "Completed",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "JobSucceeded",
		Message:            "Quantum simulation completed successfully",
	}
	r.setJobCondition(&job.Status.Conditions, condition)

	r.addJobEvent(job, "Normal", "JobSucceeded", "Quantum simulation completed successfully")

	logger.Info("Job completed successfully", "job", job.Name, "executionTime", job.Status.ExecutionTimeSec)

	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job status after success")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// handleJobFailure processes failed job and implements retry logic
func (r *QuantumJobReconciler) handleJobFailure(ctx context.Context, job *quantumv1alpha1.QuantumJob, pod *corev1.Pod) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get failure reason from pod
	reason := "Unknown"
	message := "Simulation pod failed"

	if len(pod.Status.ContainerStatuses) > 0 {
		containerStatus := pod.Status.ContainerStatuses[0]
		if containerStatus.State.Terminated != nil {
			reason = containerStatus.State.Terminated.Reason
			message = containerStatus.State.Terminated.Message
		}
	}

	// Check retry policy
	maxRetries := job.Spec.Scheduling.RetryPolicy.MaxRetries
	if maxRetries == 0 {
		maxRetries = 2 // Default
	}

	if job.Status.RetryCount < maxRetries {
		// Retry the job
		job.Status.RetryCount++
		job.Status.Phase = quantumv1alpha1.JobPhaseScheduling // Re-schedule
		job.Status.AssignedNode = ""                          // Clear node assignment

		r.addJobEvent(job, "Warning", "JobRetrying",
			fmt.Sprintf("Job failed (%s), retrying (%d/%d)", reason, job.Status.RetryCount, maxRetries))

		logger.Info("Retrying failed job", "job", job.Name, "retry", job.Status.RetryCount, "maxRetries", maxRetries)

		// Delete the failed pod
		if err := r.Delete(ctx, pod); err != nil {
			logger.Error(err, "Failed to delete failed pod")
		}

		if requeue, err := r.updateStatus(ctx, job); err != nil {
			logger.Error(err, "Failed to update job status for retry")
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}

		// Requeue with backoff
		backoff := time.Duration(job.Spec.Scheduling.RetryPolicy.BackoffSeconds) * time.Second
		if backoff == 0 {
			backoff = time.Second * 30 // Default
		}
		return ctrl.Result{RequeueAfter: backoff}, nil
	}

	// Max retries exceeded, fail the job permanently
	return r.failJob(ctx, job, fmt.Sprintf("Job failed after %d retries: %s", maxRetries, message))
}

// failJob marks the job as failed
func (r *QuantumJobReconciler) failJob(ctx context.Context, job *quantumv1alpha1.QuantumJob, errorMessage string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	now := metav1.NewTime(time.Now())
	job.Status.Phase = quantumv1alpha1.JobPhaseFailed
	job.Status.CompletionTime = &now
	job.Status.ErrorMessage = errorMessage

	// Set failed condition
	condition := metav1.Condition{
		Type:               "Completed",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: now,
		Reason:             "JobFailed",
		Message:            errorMessage,
	}
	r.setJobCondition(&job.Status.Conditions, condition)

	r.addJobEvent(job, "Warning", "JobFailed", errorMessage)

	logger.Error(fmt.Errorf("job failed"), "Job failed permanently", "job", job.Name, "error", errorMessage)

	if requeue, err := r.updateStatus(ctx, job); err != nil {
		logger.Error(err, "Failed to update job status after failure")
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// handleCompletedJob handles completed (succeeded or failed) jobs
func (r *QuantumJobReconciler) handleCompletedJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	// Job is completed, no further action needed
	// In production, you might want to implement cleanup logic here
	return ctrl.Result{}, nil
}

// handleCancelledJob handles cancelled jobs
func (r *QuantumJobReconciler) handleCancelledJob(ctx context.Context, job *quantumv1alpha1.QuantumJob) (ctrl.Result, error) {
	// TODO: Implement job cancellation logic
	// - Stop running pods
	// - Clean up resources
	return ctrl.Result{}, nil
}

// validateJobSpec validates the job specification
func (r *QuantumJobReconciler) validateJobSpec(job *quantumv1alpha1.QuantumJob) error {
	if job.Spec.UserID == "" {
		return fmt.Errorf("userID is required")
	}
	if job.Spec.Circuit.Source == "" {
		return fmt.Errorf("circuit source code is required")
	}
	if job.Spec.Scheduling.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// addJobEvent adds an event to the job's event list
func (r *QuantumJobReconciler) addJobEvent(job *quantumv1alpha1.QuantumJob, eventType, reason, message string) {
	event := quantumv1alpha1.JobEvent{
		Timestamp: metav1.NewTime(time.Now()),
		Type:      eventType,
		Reason:    reason,
		Message:   message,
	}
	job.Status.Events = append(job.Status.Events, event)

	// Keep only the last 10 events
	if len(job.Status.Events) > 10 {
		job.Status.Events = job.Status.Events[len(job.Status.Events)-10:]
	}
}

// setJobCondition sets or updates a condition in the job status
func (r *QuantumJobReconciler) setJobCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	for i, condition := range *conditions {
		if condition.Type == newCondition.Type {
			(*conditions)[i] = newCondition
			return
		}
	}
	// Condition not found, append it
	*conditions = append(*conditions, newCondition)
}

// updateStatus updates the job status with conflict retry handling.
// Returns (requeue, error). If requeue is true, caller should return Requeue.
func (r *QuantumJobReconciler) updateStatus(ctx context.Context, job *quantumv1alpha1.QuantumJob) (bool, error) {
	if err := r.Status().Update(ctx, job); err != nil {
		if errors.IsConflict(err) {
			log.FromContext(ctx).Info("Conflict updating job status, requeueing", "job", job.Name)
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *QuantumJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize internal components
	if r.PodBuilder == nil {
		r.PodBuilder = quantumruntime.NewPodBuilder()
	}
	if r.NodeScorer == nil {
		r.NodeScorer = scheduler.NewNodeScorer()
	}
	if r.Predicates == nil {
		r.Predicates = scheduler.NewPredicateRegistry()
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&quantumv1alpha1.QuantumJob{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
