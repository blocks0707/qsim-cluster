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

package scheduler

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

// NodeFilterResult represents the result of node filtering
type NodeFilterResult struct {
	NodeName string
	Reason   string
	Passed   bool
}

// Predicate defines a node filtering function
type Predicate func(*quantumv1alpha1.QuantumJob, *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult

// PredicateRegistry holds all available predicates
type PredicateRegistry struct {
	predicates []Predicate
}

// NewPredicateRegistry creates a new predicate registry with default predicates
func NewPredicateRegistry() *PredicateRegistry {
	return &PredicateRegistry{
		predicates: []Predicate{
			ResourceFitPredicate,
			PoolMatchPredicate,
			ConcurrencyLimitPredicate,
			SimulationMethodSupportPredicate,
			NodeReadyPredicate,
		},
	}
}

// Filter applies all predicates to filter suitable nodes
func (pr *PredicateRegistry) Filter(job *quantumv1alpha1.QuantumJob, nodes []*quantumv1alpha1.QuantumNodeProfile) ([]*quantumv1alpha1.QuantumNodeProfile, []NodeFilterResult) {
	var filteredNodes []*quantumv1alpha1.QuantumNodeProfile
	var filterResults []NodeFilterResult

	for _, node := range nodes {
		passed := true

		// Apply all predicates
		for _, predicate := range pr.predicates {
			result := predicate(job, node)
			filterResults = append(filterResults, *result)

			if !result.Passed {
				passed = false
				break // Stop on first failure
			}
		}

		if passed {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes, filterResults
}

// AddPredicate adds a custom predicate
func (pr *PredicateRegistry) AddPredicate(predicate Predicate) {
	pr.predicates = append(pr.predicates, predicate)
}

// ResourceFitPredicate checks if node has sufficient resources
func ResourceFitPredicate(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
	}

	// Parse required CPU
	requiredCPU, err := resource.ParseQuantity(job.Spec.Resources.CPU)
	if err != nil {
		result.Passed = false
		result.Reason = fmt.Sprintf("invalid CPU quantity: %v", err)
		return result
	}

	// Parse required memory
	requiredMemory, err := resource.ParseQuantity(job.Spec.Resources.Memory)
	if err != nil {
		result.Passed = false
		result.Reason = fmt.Sprintf("invalid memory quantity: %v", err)
		return result
	}

	// Check CPU capacity
	availableCPU := int64(node.Spec.CPU.Cores)
	if requiredCPU.Value() > availableCPU {
		result.Passed = false
		result.Reason = fmt.Sprintf("insufficient CPU: required=%d, available=%d",
			requiredCPU.Value(), availableCPU)
		return result
	}

	// Check memory capacity
	availableMemoryBytes := int64(node.Spec.Memory.TotalGB) * 1024 * 1024 * 1024 // Convert GB to bytes
	if requiredMemory.Value() > availableMemoryBytes {
		result.Passed = false
		result.Reason = fmt.Sprintf("insufficient memory: required=%s, available=%dGB",
			requiredMemory.String(), node.Spec.Memory.TotalGB)
		return result
	}

	// Check GPU requirement
	if job.Spec.Resources.GPU != "" && job.Spec.Resources.GPU != "0" {
		if !node.Spec.GPU.Available {
			result.Passed = false
			result.Reason = "GPU required but not available on node"
			return result
		}

		requiredGPU, err := strconv.Atoi(job.Spec.Resources.GPU)
		if err != nil {
			result.Passed = false
			result.Reason = fmt.Sprintf("invalid GPU quantity: %v", err)
			return result
		}

		if requiredGPU > int(node.Spec.GPU.Count) {
			result.Passed = false
			result.Reason = fmt.Sprintf("insufficient GPU: required=%d, available=%d",
				requiredGPU, node.Spec.GPU.Count)
			return result
		}
	}

	result.Reason = "resource requirements satisfied"
	return result
}

// PoolMatchPredicate checks if node pool matches job requirement
func PoolMatchPredicate(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
	}

	// Auto pool matches any node
	if job.Spec.Scheduling.NodePool == quantumv1alpha1.NodePoolAuto {
		result.Reason = "auto pool matches any node"
		return result
	}

	// Check exact pool match
	if job.Spec.Scheduling.NodePool != node.Spec.Pool {
		result.Passed = false
		result.Reason = fmt.Sprintf("pool mismatch: required=%s, node=%s",
			job.Spec.Scheduling.NodePool, node.Spec.Pool)
		return result
	}

	result.Reason = "pool requirement satisfied"
	return result
}

// ConcurrencyLimitPredicate checks if node can accept more jobs
func ConcurrencyLimitPredicate(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
	}

	maxConcurrent := node.Spec.SimulatorConfig.MaxConcurrentJobs
	if maxConcurrent == 0 {
		maxConcurrent = 3 // Default value
	}

	currentJobs := int32(0)
	if node.Status.CurrentLoad != nil {
		currentJobs = node.Status.CurrentLoad.ActiveJobs
	}

	if currentJobs >= maxConcurrent {
		result.Passed = false
		result.Reason = fmt.Sprintf("max concurrent jobs reached: %d/%d",
			currentJobs, maxConcurrent)
		return result
	}

	result.Reason = fmt.Sprintf("concurrency limit ok: %d/%d", currentJobs, maxConcurrent)
	return result
}

// SimulationMethodSupportPredicate checks if node supports required simulation method
func SimulationMethodSupportPredicate(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
	}

	// If no complexity info, allow scheduling
	if job.Spec.Complexity == nil {
		result.Reason = "no complexity info, method check skipped"
		return result
	}

	requiredMethod := job.Spec.Complexity.Method

	// Automatic method is always supported
	if requiredMethod == quantumv1alpha1.SimulationMethodAutomatic {
		result.Reason = "automatic method is always supported"
		return result
	}

	// Check if node supports the specific method
	supportedMethods := node.Spec.SimulatorConfig.SupportedMethods

	// If no methods specified, assume all are supported
	if len(supportedMethods) == 0 {
		result.Reason = "no method restrictions on node"
		return result
	}

	// Check if required method is in supported list
	for _, method := range supportedMethods {
		if method == requiredMethod {
			result.Reason = fmt.Sprintf("method %s is supported", requiredMethod)
			return result
		}
	}

	result.Passed = false
	result.Reason = fmt.Sprintf("method %s not supported on node", requiredMethod)
	return result
}

// NodeReadyPredicate checks if node is ready to accept jobs
func NodeReadyPredicate(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
	}

	// Check ready status
	if node.Status.CurrentLoad == nil {
		result.Passed = false
		result.Reason = "node status not available"
		return result
	}

	if !node.Status.Ready {
		result.Passed = false
		result.Reason = "node not ready"
		return result
	}

	result.Reason = "node is ready"
	return result
}

// PriorityBasedPredicate filters nodes based on job priority and node availability
func PriorityBasedPredicate(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
	}

	// High priority jobs can preempt resources if needed
	if job.Spec.Scheduling.Priority == quantumv1alpha1.JobPriorityHigh ||
		job.Spec.Scheduling.Priority == quantumv1alpha1.JobPriorityCritical {
		result.Reason = "high priority job can use any available node"
		return result
	}

	// Low priority jobs should avoid heavily loaded nodes
	if job.Spec.Scheduling.Priority == quantumv1alpha1.JobPriorityLow {
		if node.Status.CurrentLoad != nil && node.Status.CurrentLoad.CPUUsagePercent > 80 {
			result.Passed = false
			result.Reason = fmt.Sprintf("node too busy for low priority job: %.1f%% CPU usage",
				node.Status.CurrentLoad.CPUUsagePercent)
			return result
		}
	}

	result.Reason = "priority requirements satisfied"
	return result
}

// CustomPredicateExample demonstrates how to create custom predicates
func CustomPredicateExample(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) *NodeFilterResult {
	result := &NodeFilterResult{
		NodeName: node.Name,
		Passed:   true,
		Reason:   "custom predicate passed",
	}

	// Add custom logic here
	// For example: check special annotations, labels, or business rules

	return result
}
