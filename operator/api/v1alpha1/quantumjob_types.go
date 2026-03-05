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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=low;normal;high;critical
type JobPriority string

const (
	JobPriorityLow      JobPriority = "low"
	JobPriorityNormal   JobPriority = "normal"
	JobPriorityHigh     JobPriority = "high"
	JobPriorityCritical JobPriority = "critical"
)

// +kubebuilder:validation:Enum=auto;cpu;high-cpu;gpu
type NodePool string

const (
	NodePoolAuto    NodePool = "auto"
	NodePoolCPU     NodePool = "cpu"
	NodePoolHighCPU NodePool = "high-cpu"
	NodePoolGPU     NodePool = "gpu"
)

// +kubebuilder:validation:Enum=python;qasm
type CodeLanguage string

const (
	CodeLanguagePython CodeLanguage = "python"
	CodeLanguageQASM   CodeLanguage = "qasm"
)

// +kubebuilder:validation:Enum=statevector;stabilizer;mps;automatic
type SimulationMethod string

const (
	SimulationMethodStatevector SimulationMethod = "statevector"
	SimulationMethodStabilizer  SimulationMethod = "stabilizer" 
	SimulationMethodMPS         SimulationMethod = "mps"
	SimulationMethodAutomatic   SimulationMethod = "automatic"
)

// +kubebuilder:validation:Enum=Pending;Analyzing;Scheduling;Running;Succeeded;Failed;Cancelled
type JobPhase string

const (
	JobPhasePending    JobPhase = "Pending"
	JobPhaseAnalyzing  JobPhase = "Analyzing"
	JobPhaseScheduling JobPhase = "Scheduling"
	JobPhaseRunning    JobPhase = "Running"
	JobPhaseSucceeded  JobPhase = "Succeeded"
	JobPhaseFailed     JobPhase = "Failed"
	JobPhaseCancelled  JobPhase = "Cancelled"
)

// CircuitSpec defines the quantum circuit to be executed
type CircuitSpec struct {
	// Source code of the quantum circuit
	// +kubebuilder:validation:Required
	Source string `json:"source"`

	// Programming language used
	// +kubebuilder:default=python
	// +optional
	Language CodeLanguage `json:"language,omitempty"`

	// Python version for execution
	// +kubebuilder:default="3.11"
	// +optional
	Version string `json:"version,omitempty"`
}

// RetryPolicy defines the retry behavior for failed jobs
type RetryPolicy struct {
	// Maximum number of retries
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:default=2
	// +optional
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// Backoff time in seconds before retry
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	// +optional
	BackoffSeconds int32 `json:"backoffSeconds,omitempty"`
}

// ComplexitySpec defines the circuit complexity metadata
type ComplexitySpec struct {
	// Number of qubits in the circuit
	// +kubebuilder:validation:Minimum=1
	Qubits int32 `json:"qubits"`

	// Circuit depth
	// +kubebuilder:validation:Minimum=1
	Depth int32 `json:"depth"`

	// Total number of gates
	// +kubebuilder:validation:Minimum=1
	GateCount int32 `json:"gateCount"`

	// Circuit parallelism factor (0.0-1.0)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	Parallelism float64 `json:"parallelism"`

	// Estimated memory requirement in MB
	// +kubebuilder:validation:Minimum=1
	EstimatedMemoryMB int32 `json:"estimatedMemoryMB"`

	// Estimated CPU cores needed
	// +kubebuilder:validation:Minimum=1
	EstimatedCPUCores int32 `json:"estimatedCPUCores"`

	// Estimated execution time in seconds
	// +kubebuilder:validation:Minimum=1
	EstimatedTimeSec int32 `json:"estimatedTimeSec"`

	// Simulation method to use
	// +kubebuilder:default=automatic
	// +optional
	Method SimulationMethod `json:"method,omitempty"`
}

// SchedulingSpec defines scheduling preferences
type SchedulingSpec struct {
	// Job priority level
	// +kubebuilder:default=normal
	// +optional
	Priority JobPriority `json:"priority,omitempty"`

	// Preferred node pool
	// +kubebuilder:default=auto
	// +optional
	NodePool NodePool `json:"nodePool,omitempty"`

	// Maximum execution time in seconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	// +kubebuilder:default=300
	// +optional
	Timeout int32 `json:"timeout,omitempty"`

	// Retry policy for failed jobs
	// +optional
	RetryPolicy RetryPolicy `json:"retryPolicy,omitempty"`
}

// ResourceSpec defines the resource requirements
type ResourceSpec struct {
	// CPU cores required
	// +kubebuilder:default="2"
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory required
	// +kubebuilder:default="4Gi"
	// +optional
	Memory string `json:"memory,omitempty"`

	// GPU count required
	// +kubebuilder:default="0"
	// +optional
	GPU string `json:"gpu,omitempty"`
}

// ResultRef points to the execution result
type ResultRef struct {
	// S3/MinIO bucket name
	Bucket string `json:"bucket"`

	// Object key/path
	Key string `json:"key"`
}

// JobEvent represents an event in the job lifecycle
type JobEvent struct {
	// Timestamp of the event
	Timestamp metav1.Time `json:"timestamp"`

	// Event type (Normal, Warning)
	Type string `json:"type"`

	// Reason for the event
	Reason string `json:"reason"`

	// Human-readable message
	Message string `json:"message"`
}

// QuantumJobSpec defines the desired state of QuantumJob
type QuantumJobSpec struct {
	// User ID who submitted the job
	// +kubebuilder:validation:Required
	UserID string `json:"userID"`

	// Circuit specification
	// +kubebuilder:validation:Required
	Circuit CircuitSpec `json:"circuit"`

	// Circuit complexity metadata (filled by analyzer)
	// +optional
	Complexity *ComplexitySpec `json:"complexity,omitempty"`

	// Scheduling preferences
	// +optional
	Scheduling SchedulingSpec `json:"scheduling,omitempty"`

	// Resource requirements
	// +optional
	Resources ResourceSpec `json:"resources,omitempty"`
}

// QuantumJobStatus defines the observed state of QuantumJob
type QuantumJobStatus struct {
	// Current phase of the job
	// +optional
	Phase JobPhase `json:"phase,omitempty"`

	// Conditions represent the latest observations of job state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Node assigned for execution
	// +optional
	AssignedNode string `json:"assignedNode,omitempty"`

	// Node pool assigned
	// +optional
	AssignedPool NodePool `json:"assignedPool,omitempty"`

	// Time when job execution started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Time when job execution completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Actual execution time in seconds
	// +optional
	ExecutionTimeSec *int32 `json:"executionTimeSec,omitempty"`

	// Reference to the execution result
	// +optional
	ResultRef *ResultRef `json:"resultRef,omitempty"`

	// Job lifecycle events
	// +optional
	Events []JobEvent `json:"events,omitempty"`

	// Error message if job failed
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`

	// Number of retry attempts made
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=quantum
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Qubits",type="integer",JSONPath=".spec.complexity.qubits"
// +kubebuilder:printcolumn:name="Depth",type="integer",JSONPath=".spec.complexity.depth"
// +kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.assignedNode"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// QuantumJob is the Schema for the quantumjobs API
type QuantumJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuantumJobSpec   `json:"spec,omitempty"`
	Status QuantumJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// QuantumJobList contains a list of QuantumJob
type QuantumJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuantumJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QuantumJob{}, &QuantumJobList{})
}