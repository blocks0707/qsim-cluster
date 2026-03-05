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
	"math"
	"sort"

	"k8s.io/apimachinery/pkg/api/resource"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

// Scoring weights from design document
const (
	ResourceFitWeight  = 0.4
	LoadBalanceWeight  = 0.3
	PoolMatchWeight    = 0.2
	LocalityWeight     = 0.1
)

// NodeScore represents the score of a node for a specific job
type NodeScore struct {
	NodeName       string
	TotalScore     float64
	ResourceFit    float64
	LoadBalance    float64
	PoolMatch      float64
	LocalityScore  float64
	Details        string
}

// NodeScorer calculates node fitness scores for quantum jobs
type NodeScorer struct {
	// Weights for different scoring factors
	Weights ScoreWeights
}

// ScoreWeights defines the weights for different scoring components
type ScoreWeights struct {
	ResourceFit  float64
	LoadBalance  float64
	PoolMatch    float64
	Locality     float64
}

// NewNodeScorer creates a new node scorer with default weights
func NewNodeScorer() *NodeScorer {
	return &NodeScorer{
		Weights: ScoreWeights{
			ResourceFit:  ResourceFitWeight,
			LoadBalance:  LoadBalanceWeight,
			PoolMatch:    PoolMatchWeight,
			Locality:     LocalityWeight,
		},
	}
}

// NewNodeScorerWithWeights creates a new node scorer with custom weights
func NewNodeScorerWithWeights(weights ScoreWeights) *NodeScorer {
	return &NodeScorer{
		Weights: weights,
	}
}

// ScoreNodes calculates scores for all nodes and returns them sorted by score (highest first)
func (ns *NodeScorer) ScoreNodes(job *quantumv1alpha1.QuantumJob, nodes []*quantumv1alpha1.QuantumNodeProfile) ([]*NodeScore, error) {
	var scores []*NodeScore
	
	for _, node := range nodes {
		score, err := ns.ScoreNode(job, node)
		if err != nil {
			// Log error but continue with other nodes
			continue
		}
		scores = append(scores, score)
	}
	
	// Sort by total score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})
	
	return scores, nil
}

// ScoreNode calculates the score for a single node
func (ns *NodeScorer) ScoreNode(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) (*NodeScore, error) {
	score := &NodeScore{
		NodeName: node.Name,
	}
	
	// Calculate individual scoring components
	resourceFit := ns.calculateResourceFit(job, node)
	loadBalance := ns.calculateLoadBalance(node)
	poolMatch := ns.calculatePoolMatch(job, node)
	locality := ns.calculateLocalityScore(job, node)
	
	// Calculate weighted total score
	totalScore := resourceFit*ns.Weights.ResourceFit +
		loadBalance*ns.Weights.LoadBalance +
		poolMatch*ns.Weights.PoolMatch +
		locality*ns.Weights.Locality
	
	score.ResourceFit = resourceFit
	score.LoadBalance = loadBalance
	score.PoolMatch = poolMatch
	score.LocalityScore = locality
	score.TotalScore = totalScore
	score.Details = ns.generateScoreDetails(score)
	
	return score, nil
}

// calculateResourceFit scores how well the node's resources match the job requirements
// Returns value between 0.0 and 1.0
func (ns *NodeScorer) calculateResourceFit(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) float64 {
	// Parse required resources
	requiredCPU, err := resource.ParseQuantity(job.Spec.Resources.CPU)
	if err != nil {
		return 0.0
	}
	
	requiredMemory, err := resource.ParseQuantity(job.Spec.Resources.Memory)
	if err != nil {
		return 0.0
	}
	
	// Calculate available resources
	availableCPU := float64(node.Spec.CPU.Cores)
	availableMemoryGB := float64(node.Spec.Memory.TotalGB)
	
	// Account for current usage
	if node.Status.CurrentLoad != nil {
		availableCPU *= (100.0 - node.Status.CurrentLoad.CPUUsagePercent) / 100.0
		availableMemoryGB *= (100.0 - node.Status.CurrentLoad.MemoryUsagePercent) / 100.0
	}
	
	// Calculate fit ratios
	cpuFit := availableCPU / float64(requiredCPU.Value())
	memoryFit := (availableMemoryGB * 1024 * 1024 * 1024) / float64(requiredMemory.Value())
	
	// Use the minimum fit ratio as the resource fit score
	resourceFit := math.Min(cpuFit, memoryFit)
	
	// Cap at 1.0 and ensure non-negative
	if resourceFit > 1.0 {
		resourceFit = 1.0
	}
	if resourceFit < 0.0 {
		resourceFit = 0.0
	}
	
	// Bonus for overprovisioned nodes (better for complex jobs)
	if job.Spec.Complexity != nil && job.Spec.Complexity.EstimatedCPUCores > 4 {
		if cpuFit > 2.0 && memoryFit > 2.0 {
			resourceFit = math.Min(1.0, resourceFit*1.2) // 20% bonus
		}
	}
	
	return resourceFit
}

// calculateLoadBalance scores how balanced the load would be after scheduling this job
// Returns value between 0.0 and 1.0 (higher = better balance)
func (ns *NodeScorer) calculateLoadBalance(node *quantumv1alpha1.QuantumNodeProfile) float64 {
	if node.Status.CurrentLoad == nil {
		return 0.5 // Neutral score if no load data
	}
	
	// Get current active jobs and max concurrent
	activeJobs := float64(node.Status.CurrentLoad.ActiveJobs)
	maxConcurrent := float64(node.Spec.SimulatorConfig.MaxConcurrentJobs)
	
	if maxConcurrent == 0 {
		maxConcurrent = 3 // Default value
	}
	
	// Calculate job utilization (0.0 to 1.0)
	jobUtilization := activeJobs / maxConcurrent
	
	// Calculate CPU utilization
	cpuUtilization := node.Status.CurrentLoad.CPUUsagePercent / 100.0
	
	// Calculate memory utilization
	memoryUtilization := node.Status.CurrentLoad.MemoryUsagePercent / 100.0
	
	// Average utilization
	avgUtilization := (jobUtilization + cpuUtilization + memoryUtilization) / 3.0
	
	// Load balance score: prefer nodes with lower utilization
	// Use inverted exponential curve to heavily penalize overloaded nodes
	loadBalanceScore := math.Exp(-2 * avgUtilization)
	
	return math.Max(0.0, math.Min(1.0, loadBalanceScore))
}

// calculatePoolMatch scores how well the node pool matches the job requirements
// Returns value between 0.0 and 1.0
func (ns *NodeScorer) calculatePoolMatch(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) float64 {
	requiredPool := job.Spec.Scheduling.NodePool
	nodePool := node.Spec.Pool
	
	// Perfect match
	if requiredPool == nodePool {
		return 1.0
	}
	
	// Auto pool gets medium score for all pools
	if requiredPool == quantumv1alpha1.NodePoolAuto {
		return 0.5
	}
	
	// Some pools are compatible with others
	switch requiredPool {
	case quantumv1alpha1.NodePoolCPU:
		if nodePool == quantumv1alpha1.NodePoolHighCPU {
			return 0.7 // High-CPU can handle CPU jobs well
		}
	case quantumv1alpha1.NodePoolHighCPU:
		if nodePool == quantumv1alpha1.NodePoolGPU {
			return 0.6 // GPU nodes often have high CPU as well
		}
	}
	
	// No match
	return 0.0
}

// calculateLocalityScore scores data locality and other node-specific advantages
// Returns value between 0.0 and 1.0
func (ns *NodeScorer) calculateLocalityScore(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) float64 {
	score := 0.5 // Base locality score
	
	// Bonus for nodes that have run similar jobs (complexity-based)
	if job.Spec.Complexity != nil {
		score += ns.calculateComplexityAffinityBonus(job, node)
	}
	
	// Bonus for GPU availability when needed
	if ns.needsGPU(job) && node.Spec.GPU.Available {
		score += 0.2
	}
	
	// Bonus for architectural match
	if node.Spec.CPU.Architecture == quantumv1alpha1.CPUArchitectureX86_64 {
		score += 0.1 // x86_64 generally has better quantum simulator support
	}
	
	// Bonus for newer/faster GPU types
	if node.Spec.GPU.Available {
		switch node.Spec.GPU.Type {
		case "A100", "H100":
			score += 0.2
		case "V100", "A40":
			score += 0.1
		}
	}
	
	return math.Max(0.0, math.Min(1.0, score))
}

// calculateComplexityAffinityBonus gives bonus score for nodes that match job complexity
func (ns *NodeScorer) calculateComplexityAffinityBonus(job *quantumv1alpha1.QuantumJob, node *quantumv1alpha1.QuantumNodeProfile) float64 {
	qubits := job.Spec.Complexity.Qubits
	method := job.Spec.Complexity.Method
	
	// Bonus for method-specific optimizations
	switch method {
	case quantumv1alpha1.SimulationMethodStatevector:
		// Statevector benefits from high memory
		if node.Spec.Memory.TotalGB >= 64 {
			return 0.1
		}
	case quantumv1alpha1.SimulationMethodMPS:
		// MPS benefits from high CPU count
		if node.Spec.CPU.Cores >= 16 {
			return 0.1
		}
	case quantumv1alpha1.SimulationMethodStabilizer:
		// Stabilizer is CPU-efficient, works well on standard nodes
		if node.Spec.Pool == quantumv1alpha1.NodePoolCPU {
			return 0.1
		}
	}
	
	// Bonus for qubit count matching node capabilities
	if qubits <= 10 && node.Spec.Pool == quantumv1alpha1.NodePoolCPU {
		return 0.1 // Small circuits work well on CPU nodes
	} else if qubits > 20 && node.Spec.Pool == quantumv1alpha1.NodePoolGPU {
		return 0.15 // Large circuits need GPU acceleration
	} else if qubits > 15 && node.Spec.Pool == quantumv1alpha1.NodePoolHighCPU {
		return 0.1 // Medium circuits work well on high-CPU nodes
	}
	
	return 0.0
}

// needsGPU determines if a job would benefit from GPU acceleration
func (ns *NodeScorer) needsGPU(job *quantumv1alpha1.QuantumJob) bool {
	// Explicit GPU requirement
	if job.Spec.Resources.GPU != "" && job.Spec.Resources.GPU != "0" {
		return true
	}
	
	// GPU recommended for large circuits
	if job.Spec.Complexity != nil && job.Spec.Complexity.Qubits > 20 {
		return true
	}
	
	// GPU helpful for statevector method with many qubits
	if job.Spec.Complexity != nil && 
	   job.Spec.Complexity.Method == quantumv1alpha1.SimulationMethodStatevector && 
	   job.Spec.Complexity.Qubits > 15 {
		return true
	}
	
	return false
}

// generateScoreDetails creates a human-readable explanation of the score
func (ns *NodeScorer) generateScoreDetails(score *NodeScore) string {
	return fmt.Sprintf("Total: %.3f (ResourceFit: %.3f, LoadBalance: %.3f, PoolMatch: %.3f, Locality: %.3f)",
		score.TotalScore, score.ResourceFit, score.LoadBalance, score.PoolMatch, score.LocalityScore)
}

// GetBestNode returns the highest-scoring node from a list
func (ns *NodeScorer) GetBestNode(job *quantumv1alpha1.QuantumJob, nodes []*quantumv1alpha1.QuantumNodeProfile) (*quantumv1alpha1.QuantumNodeProfile, *NodeScore, error) {
	scores, err := ns.ScoreNodes(job, nodes)
	if err != nil {
		return nil, nil, err
	}
	
	if len(scores) == 0 {
		return nil, nil, fmt.Errorf("no nodes available for scoring")
	}
	
	// Find the corresponding node for the best score
	bestScore := scores[0]
	for _, node := range nodes {
		if node.Name == bestScore.NodeName {
			return node, bestScore, nil
		}
	}
	
	return nil, nil, fmt.Errorf("best scoring node not found in node list")
}