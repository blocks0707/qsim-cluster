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

// +kubebuilder:validation:Enum=x86_64;arm64
type CPUArchitecture string

const (
	CPUArchitectureX86_64 CPUArchitecture = "x86_64"
	CPUArchitectureARM64  CPUArchitecture = "arm64"
)

// CPUCapabilities defines CPU capabilities of the node
type CPUCapabilities struct {
	// Number of CPU cores
	// +kubebuilder:validation:Minimum=1
	Cores int32 `json:"cores"`

	// CPU architecture
	// +kubebuilder:default=x86_64
	// +optional
	Architecture CPUArchitecture `json:"architecture,omitempty"`
}

// MemoryCapabilities defines memory capabilities of the node
type MemoryCapabilities struct {
	// Total memory in GB
	// +kubebuilder:validation:Minimum=1
	TotalGB int32 `json:"totalGB"`
}

// GPUCapabilities defines GPU capabilities of the node
type GPUCapabilities struct {
	// Whether GPU is available
	Available bool `json:"available"`

	// GPU type/model (e.g., "A100", "V100")
	// +optional
	Type string `json:"type,omitempty"`

	// Number of GPUs
	// +optional
	Count int32 `json:"count,omitempty"`

	// GPU memory in GB
	// +optional
	MemoryGB int32 `json:"memoryGB,omitempty"`
}

// SimulatorConfig defines quantum simulator configuration for the node
type SimulatorConfig struct {
	// Maximum concurrent jobs this node can handle
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	// +optional
	MaxConcurrentJobs int32 `json:"maxConcurrentJobs,omitempty"`

	// Supported simulation methods
	// +optional
	SupportedMethods []SimulationMethod `json:"supportedMethods,omitempty"`
}

// LoadStatus represents current resource usage
type LoadStatus struct {
	// Current CPU usage percentage (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	CPUUsagePercent float64 `json:"cpuUsagePercent"`

	// Current memory usage percentage (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MemoryUsagePercent float64 `json:"memoryUsagePercent"`

	// Current number of active quantum jobs
	// +kubebuilder:validation:Minimum=0
	ActiveJobs int32 `json:"activeJobs"`
}

// QuantumNodeProfileSpec defines the desired state of QuantumNodeProfile
type QuantumNodeProfileSpec struct {
	// Node pool this node belongs to
	// +kubebuilder:validation:Required
	Pool NodePool `json:"pool"`

	// CPU capabilities
	// +kubebuilder:validation:Required
	CPU CPUCapabilities `json:"cpu"`

	// Memory capabilities
	// +kubebuilder:validation:Required
	Memory MemoryCapabilities `json:"memory"`

	// GPU capabilities
	// +kubebuilder:validation:Required
	GPU GPUCapabilities `json:"gpu"`

	// Quantum simulator configuration
	// +optional
	SimulatorConfig SimulatorConfig `json:"simulatorConfig,omitempty"`
}

// QuantumNodeProfileStatus defines the observed state of QuantumNodeProfile
type QuantumNodeProfileStatus struct {
	// Current resource load and usage
	// +optional
	CurrentLoad *LoadStatus `json:"currentLoad,omitempty"`

	// Last time the status was updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Whether the node is ready to accept quantum jobs
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions represent the latest observations of node state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=quantum
// +kubebuilder:printcolumn:name="Pool",type="string",JSONPath=".spec.pool"
// +kubebuilder:printcolumn:name="CPU",type="integer",JSONPath=".spec.cpu.cores"
// +kubebuilder:printcolumn:name="Memory",type="string",JSONPath=".spec.memory.totalGB"
// +kubebuilder:printcolumn:name="GPU",type="boolean",JSONPath=".spec.gpu.available"
// +kubebuilder:printcolumn:name="Load",type="string",JSONPath=".status.currentLoad.cpuUsagePercent"
// +kubebuilder:printcolumn:name="Active Jobs",type="integer",JSONPath=".status.currentLoad.activeJobs"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// QuantumNodeProfile is the Schema for the quantumnodeprofiles API
type QuantumNodeProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuantumNodeProfileSpec   `json:"spec,omitempty"`
	Status QuantumNodeProfileStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// QuantumNodeProfileList contains a list of QuantumNodeProfile
type QuantumNodeProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuantumNodeProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QuantumNodeProfile{}, &QuantumNodeProfileList{})
}
