package scheduler

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceFitPredicate(t *testing.T) {
	tests := []struct {
		name               string
		job                *quantumv1alpha1.QuantumJob
		node               *quantumv1alpha1.QuantumNodeProfile
		wantPassed         bool
		wantReasonContains string
	}{
		{
			name: "sufficient resources",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{CPU: "2", Memory: "4Gi"},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
					Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
				},
			},
			wantPassed:         true,
			wantReasonContains: "satisfied",
		},
		{
			name: "insufficient CPU",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{CPU: "8", Memory: "2Gi"},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
					Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
				},
			},
			wantPassed:         false,
			wantReasonContains: "insufficient CPU",
		},
		{
			name: "insufficient memory",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{CPU: "2", Memory: "16Gi"},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
					Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
				},
			},
			wantPassed:         false,
			wantReasonContains: "insufficient memory",
		},
		{
			name: "GPU required but not available",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{CPU: "2", Memory: "4Gi", GPU: "1"},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
					Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
					GPU:    quantumv1alpha1.GPUCapabilities{Available: false},
				},
			},
			wantPassed:         false,
			wantReasonContains: "GPU",
		},
		{
			name: "GPU available and sufficient",
			job: &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{CPU: "2", Memory: "4Gi", GPU: "1"},
				},
			},
			node: &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
					Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
					GPU:    quantumv1alpha1.GPUCapabilities{Available: true, Count: 2},
				},
			},
			wantPassed:         true,
			wantReasonContains: "satisfied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResourceFitPredicate(tt.job, tt.node)
			assert.Equal(t, tt.node.Name, result.NodeName)
			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Contains(t, result.Reason, tt.wantReasonContains)
		})
	}
}

func TestPoolMatchPredicate(t *testing.T) {
	tests := []struct {
		name               string
		jobPool            quantumv1alpha1.NodePool
		nodePool           quantumv1alpha1.NodePool
		wantPassed         bool
		wantReasonContains string
	}{
		{"auto matches any", quantumv1alpha1.NodePoolAuto, quantumv1alpha1.NodePoolCPU, true, "auto"},
		{"exact match", quantumv1alpha1.NodePoolGPU, quantumv1alpha1.NodePoolGPU, true, "satisfied"},
		{"mismatch", quantumv1alpha1.NodePoolGPU, quantumv1alpha1.NodePoolCPU, false, "mismatch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Scheduling: quantumv1alpha1.SchedulingSpec{NodePool: tt.jobPool},
				},
			}
			node := &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec:       quantumv1alpha1.QuantumNodeProfileSpec{Pool: tt.nodePool},
			}
			result := PoolMatchPredicate(job, node)
			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Contains(t, result.Reason, tt.wantReasonContains)
		})
	}
}

func TestConcurrencyLimitPredicate(t *testing.T) {
	tests := []struct {
		name       string
		maxJobs    int32
		activeJobs int32
		wantPassed bool
	}{
		{"within limit", 5, 3, true},
		{"at limit", 3, 3, false},
		{"default limit ok", 0, 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					SimulatorConfig: quantumv1alpha1.SimulatorConfig{
						MaxConcurrentJobs: tt.maxJobs,
					},
				},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					CurrentLoad: &quantumv1alpha1.LoadStatus{ActiveJobs: tt.activeJobs},
				},
			}
			result := ConcurrencyLimitPredicate(&quantumv1alpha1.QuantumJob{}, node)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

func TestSimulationMethodSupportPredicate(t *testing.T) {
	tests := []struct {
		name       string
		method     quantumv1alpha1.SimulationMethod
		supported  []quantumv1alpha1.SimulationMethod
		wantPassed bool
	}{
		{"automatic always passes", quantumv1alpha1.SimulationMethodAutomatic, nil, true},
		{"supported method", quantumv1alpha1.SimulationMethodStatevector,
			[]quantumv1alpha1.SimulationMethod{quantumv1alpha1.SimulationMethodStatevector}, true},
		{"unsupported method", quantumv1alpha1.SimulationMethodMPS,
			[]quantumv1alpha1.SimulationMethod{quantumv1alpha1.SimulationMethodStatevector}, false},
		{"no restrictions", quantumv1alpha1.SimulationMethodMPS, []quantumv1alpha1.SimulationMethod{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Complexity: &quantumv1alpha1.ComplexitySpec{Method: tt.method},
				},
			}
			node := &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					SimulatorConfig: quantumv1alpha1.SimulatorConfig{SupportedMethods: tt.supported},
				},
			}
			result := SimulationMethodSupportPredicate(job, node)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

func TestNodeReadyPredicate(t *testing.T) {
	tests := []struct {
		name       string
		ready      bool
		hasLoad    bool
		wantPassed bool
	}{
		{"ready with load", true, true, true},
		{"not ready", false, true, false},
		{"no load status", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{Name: "node1"},
				Status: quantumv1alpha1.QuantumNodeProfileStatus{
					Ready: tt.ready,
				},
			}
			if tt.hasLoad {
				node.Status.CurrentLoad = &quantumv1alpha1.LoadStatus{ActiveJobs: 1}
			}
			result := NodeReadyPredicate(&quantumv1alpha1.QuantumJob{}, node)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

func TestPredicateRegistry_Filter(t *testing.T) {
	job := &quantumv1alpha1.QuantumJob{
		Spec: quantumv1alpha1.QuantumJobSpec{
			Resources:  quantumv1alpha1.ResourceSpec{CPU: "2", Memory: "4Gi"},
			Scheduling: quantumv1alpha1.SchedulingSpec{NodePool: quantumv1alpha1.NodePoolCPU},
			Complexity: &quantumv1alpha1.ComplexitySpec{Method: quantumv1alpha1.SimulationMethodStatevector},
		},
	}

	nodes := []*quantumv1alpha1.QuantumNodeProfile{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "good-node"},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool:   quantumv1alpha1.NodePoolCPU,
				CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
				Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
				SimulatorConfig: quantumv1alpha1.SimulatorConfig{
					MaxConcurrentJobs: 5,
					SupportedMethods:  []quantumv1alpha1.SimulationMethod{quantumv1alpha1.SimulationMethodStatevector},
				},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				Ready:       true,
				CurrentLoad: &quantumv1alpha1.LoadStatus{ActiveJobs: 2},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "insufficient-cpu"},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool:   quantumv1alpha1.NodePoolCPU,
				CPU:    quantumv1alpha1.CPUCapabilities{Cores: 1},
				Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				Ready:       true,
				CurrentLoad: &quantumv1alpha1.LoadStatus{ActiveJobs: 0},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "wrong-pool"},
			Spec: quantumv1alpha1.QuantumNodeProfileSpec{
				Pool:   quantumv1alpha1.NodePoolGPU,
				CPU:    quantumv1alpha1.CPUCapabilities{Cores: 4},
				Memory: quantumv1alpha1.MemoryCapabilities{TotalGB: 8},
			},
			Status: quantumv1alpha1.QuantumNodeProfileStatus{
				Ready:       true,
				CurrentLoad: &quantumv1alpha1.LoadStatus{ActiveJobs: 0},
			},
		},
	}

	registry := NewPredicateRegistry()
	filteredNodes, results := registry.Filter(job, nodes)

	require.Len(t, filteredNodes, 1)
	assert.Equal(t, "good-node", filteredNodes[0].Name)
	assert.Greater(t, len(results), 0)
}
