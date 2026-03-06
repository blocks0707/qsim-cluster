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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

func TestNodeScorer_CalculateResourceFit(t *testing.T) {
	scorer := NewNodeScorer()

	testCases := []struct {
		name          string
		job           *quantumv1alpha1.QuantumJob
		node          *quantumv1alpha1.QuantumNodeProfile
		expectedScore float64
		expectError   bool
	}{
		{
			name: "perfect fit",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "2",
						Memory: "4Gi",
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU: quantumv1alpha1.CPUCapabilities{
						Cores: 4,
					},
					Memory: quantumv1alpha1.MemoryCapabilities{
						TotalGB: 8,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: &quantumv1alpha1.LoadStatus{
						CPUUsagePercent:    0,
						MemoryUsagePercent: 0,
						ActiveJobs:         0,
					},
				},
			},
			expectedScore: 1.0,
		},
		{
			name: "insufficient resources",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "8",
						Memory: "16Gi",
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU: quantumv1alpha1.CPUCapabilities{
						Cores: 4,
					},
					Memory: quantumv1alpha1.MemoryCapabilities{
						TotalGB: 8,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: &quantumv1alpha1.LoadStatus{
						CPUUsagePercent:    0,
						MemoryUsagePercent: 0,
						ActiveJobs:         0,
					},
				},
			},
			expectedScore: 0.5, // min(4/8, 8GB/16GB) = 0.5, capped at resource fit ratio
		},
		{
			name: "partial load",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "2",
						Memory: "4Gi",
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU: quantumv1alpha1.CPUCapabilities{
						Cores: 4,
					},
					Memory: quantumv1alpha1.MemoryCapabilities{
						TotalGB: 8,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: &quantumv1alpha1.LoadStatus{
						CPUUsagePercent:    50,
						MemoryUsagePercent: 50,
						ActiveJobs:         1,
					},
				},
			},
			expectedScore: 1.0, // Still fits after accounting for current usage
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scorer.calculateResourceFit(tc.job, tc.node)

			if score != tc.expectedScore {
				t.Errorf("Expected score %f, got %f", tc.expectedScore, score)
			}
		})
	}
}

func TestNodeScorer_CalculateLoadBalance(t *testing.T) {
	scorer := NewNodeScorer()

	testCases := []struct {
		name          string
		node          *quantumv1alpha1.QuantumNodeProfile
		expectedRange [2]float64 // min, max expected score
	}{
		{
			name: "low load",
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					SimulatorConfig: quantumv1alpha1.SimulatorConfig{
						MaxConcurrentJobs: 3,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: &quantumv1alpha1.LoadStatus{
						CPUUsagePercent:    10,
						MemoryUsagePercent: 20,
						ActiveJobs:         1,
					},
				},
			},
			expectedRange: [2]float64{0.5, 1.0},
		},
		{
			name: "high load",
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					SimulatorConfig: quantumv1alpha1.SimulatorConfig{
						MaxConcurrentJobs: 3,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: &quantumv1alpha1.LoadStatus{
						CPUUsagePercent:    90,
						MemoryUsagePercent: 85,
						ActiveJobs:         3,
					},
				},
			},
			expectedRange: [2]float64{0.0, 0.3},
		},
		{
			name: "no load info",
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					SimulatorConfig: quantumv1alpha1.SimulatorConfig{
						MaxConcurrentJobs: 3,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: nil,
				},
			},
			expectedRange: [2]float64{0.5, 0.5}, // Should return 0.5 for no data
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scorer.calculateLoadBalance(tc.node)

			if score < tc.expectedRange[0] || score > tc.expectedRange[1] {
				t.Errorf("Expected score between %f and %f, got %f",
					tc.expectedRange[0], tc.expectedRange[1], score)
			}
		})
	}
}

func TestNodeScorer_CalculatePoolMatch(t *testing.T) {
	scorer := NewNodeScorer()

	testCases := []struct {
		name          string
		job           *quantumv1alpha1.QuantumJob
		node          *quantumv1alpha1.QuantumNodeProfile
		expectedScore float64
	}{
		{
			name: "exact pool match",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Scheduling: quantumv1alpha1.SchedulingSpec{
						NodePool: quantumv1alpha1.NodePoolCPU,
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					Pool: quantumv1alpha1.NodePoolCPU,
				},
			},
			expectedScore: 1.0,
		},
		{
			name: "auto pool",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Scheduling: quantumv1alpha1.SchedulingSpec{
						NodePool: quantumv1alpha1.NodePoolAuto,
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					Pool: quantumv1alpha1.NodePoolGPU,
				},
			},
			expectedScore: 0.5,
		},
		{
			name: "compatible pools (CPU -> High-CPU)",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Scheduling: quantumv1alpha1.SchedulingSpec{
						NodePool: quantumv1alpha1.NodePoolCPU,
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					Pool: quantumv1alpha1.NodePoolHighCPU,
				},
			},
			expectedScore: 0.7,
		},
		{
			name: "incompatible pools",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Scheduling: quantumv1alpha1.SchedulingSpec{
						NodePool: quantumv1alpha1.NodePoolGPU,
					},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					Pool: quantumv1alpha1.NodePoolCPU,
				},
			},
			expectedScore: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := scorer.calculatePoolMatch(tc.job, tc.node)

			if score != tc.expectedScore {
				t.Errorf("Expected score %f, got %f", tc.expectedScore, score)
			}
		})
	}
}

func TestNodeScorer_ScoreNodes(t *testing.T) {
	scorer := NewNodeScorer()

	// Create test job
	job := &quantumv1alpha1.QuantumJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
		Spec: quantumv1alpha1.QuantumJobSpec{
			Resources: quantumv1alpha1.ResourceSpec{
				CPU:    "2",
				Memory: "4Gi",
			},
			Scheduling: quantumv1alpha1.SchedulingSpec{
				NodePool: quantumv1alpha1.NodePoolCPU,
			},
			Complexity: &quantumv1alpha1.ComplexitySpec{
				Qubits:            10,
				Method:            quantumv1alpha1.SimulationMethodStatevector,
				EstimatedCPUCores: 2,
			},
		},
	}

	// Create test nodes with different characteristics
	nodes := []*quantumv1alpha1.QuantumNodeProfile{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1-perfect",
			},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool: quantumv1alpha1.NodePoolCPU,
				CPU: quantumv1alpha1.CPUCapabilities{
					Cores: 4,
				},
				Memory: quantumv1alpha1.MemoryCapabilities{
					TotalGB: 8,
				},
				SimulatorConfig: quantumv1alpha1.SimulatorConfig{
					MaxConcurrentJobs: 3,
				},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				CurrentLoad: &quantumv1alpha1.LoadStatus{
					CPUUsagePercent:    10,
					MemoryUsagePercent: 10,
					ActiveJobs:         0,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2-busy",
			},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool: quantumv1alpha1.NodePoolCPU,
				CPU: quantumv1alpha1.CPUCapabilities{
					Cores: 4,
				},
				Memory: quantumv1alpha1.MemoryCapabilities{
					TotalGB: 8,
				},
				SimulatorConfig: quantumv1alpha1.SimulatorConfig{
					MaxConcurrentJobs: 3,
				},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				CurrentLoad: &quantumv1alpha1.LoadStatus{
					CPUUsagePercent:    80,
					MemoryUsagePercent: 70,
					ActiveJobs:         2,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-3-wrong-pool",
			},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool: quantumv1alpha1.NodePoolGPU,
				CPU: quantumv1alpha1.CPUCapabilities{
					Cores: 8,
				},
				Memory: quantumv1alpha1.MemoryCapabilities{
					TotalGB: 16,
				},
				GPU: quantumv1alpha1.GPUCapabilities{
					Available: true,
					Type:      "A100",
					Count:     1,
				},
				SimulatorConfig: quantumv1alpha1.SimulatorConfig{
					MaxConcurrentJobs: 2,
				},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				CurrentLoad: &quantumv1alpha1.LoadStatus{
					CPUUsagePercent:    5,
					MemoryUsagePercent: 5,
					ActiveJobs:         0,
				},
			},
		},
	}

	// Score all nodes
	scores, err := scorer.ScoreNodes(job, nodes)
	if err != nil {
		t.Fatalf("Failed to score nodes: %v", err)
	}

	if len(scores) != 3 {
		t.Fatalf("Expected 3 scores, got %d", len(scores))
	}

	// Verify scores are sorted (highest first)
	for i := 1; i < len(scores); i++ {
		if scores[i-1].TotalScore < scores[i].TotalScore {
			t.Errorf("Scores not sorted: score[%d]=%.3f > score[%d]=%.3f",
				i-1, scores[i-1].TotalScore, i, scores[i].TotalScore)
		}
	}

	// The perfect node should score highest
	if scores[0].NodeName != "node-1-perfect" {
		t.Errorf("Expected node-1-perfect to score highest, got %s", scores[0].NodeName)
	}

	// Print scores for debugging
	t.Logf("Scoring results:")
	for _, score := range scores {
		t.Logf("  %s: %.3f (%s)", score.NodeName, score.TotalScore, score.Details)
	}
}

func TestNodeScorer_GetBestNode(t *testing.T) {
	scorer := NewNodeScorer()

	job := &quantumv1alpha1.QuantumJob{
		Spec: quantumv1alpha1.QuantumJobSpec{
			Resources: quantumv1alpha1.ResourceSpec{
				CPU:    "2",
				Memory: "4Gi",
			},
			Scheduling: quantumv1alpha1.SchedulingSpec{
				NodePool: quantumv1alpha1.NodePoolAuto,
			},
		},
	}

	nodes := []*quantumv1alpha1.QuantumNodeProfile{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "best-node",
			},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool: quantumv1alpha1.NodePoolCPU,
				CPU: quantumv1alpha1.CPUCapabilities{
					Cores: 8,
				},
				Memory: quantumv1alpha1.MemoryCapabilities{
					TotalGB: 16,
				},
				SimulatorConfig: quantumv1alpha1.SimulatorConfig{
					MaxConcurrentJobs: 5,
				},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				CurrentLoad: &quantumv1alpha1.LoadStatus{
					CPUUsagePercent:    10,
					MemoryUsagePercent: 10,
					ActiveJobs:         0,
				},
			},
		},
	}

	bestNode, bestScore, err := scorer.GetBestNode(job, nodes)
	if err != nil {
		t.Fatalf("Failed to get best node: %v", err)
	}

	if bestNode.Name != "best-node" {
		t.Errorf("Expected best node to be 'best-node', got %s", bestNode.Name)
	}

	if bestScore.TotalScore <= 0 {
		t.Errorf("Expected positive score, got %.3f", bestScore.TotalScore)
	}

	t.Logf("Best node: %s with score %.3f (%s)",
		bestNode.Name, bestScore.TotalScore, bestScore.Details)
}
