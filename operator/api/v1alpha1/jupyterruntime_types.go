/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JupyterRuntime lifecycle phases
const (
	JupyterPhasePending     = "Pending"
	JupyterPhaseProvisioning = "Provisioning"
	JupyterPhaseRunning     = "Running"
	JupyterPhaseStopping    = "Stopping"
	JupyterPhaseStopped     = "Stopped"
	JupyterPhaseFailed      = "Failed"
)

// JupyterRuntimeSpec defines the desired state of JupyterRuntime
type JupyterRuntimeSpec struct {
	// UserID is the owner of this Jupyter session
	UserID string `json:"userID"`

	// Resources defines compute resources for the notebook
	Resources JupyterResources `json:"resources,omitempty"`

	// Image is the container image for the Jupyter server
	// +kubebuilder:default="jupyter/scipy-notebook:latest"
	Image string `json:"image,omitempty"`

	// Timeout is the idle timeout in seconds before auto-shutdown
	// +kubebuilder:default=3600
	// +kubebuilder:validation:Minimum=300
	Timeout int32 `json:"timeout,omitempty"`

	// Packages lists additional pip packages to install on startup
	Packages []string `json:"packages,omitempty"`

	// QSimEndpoint is the API server endpoint for SDK access
	QSimEndpoint string `json:"qsimEndpoint,omitempty"`
}

// JupyterResources defines resources for a Jupyter notebook
type JupyterResources struct {
	// +kubebuilder:default="2"
	CPU string `json:"cpu,omitempty"`
	// +kubebuilder:default="4Gi"
	Memory string `json:"memory,omitempty"`
	// +kubebuilder:default="10Gi"
	Storage string `json:"storage,omitempty"`
}

// JupyterRuntimeStatus defines the observed state of JupyterRuntime
type JupyterRuntimeStatus struct {
	// Phase is the current lifecycle phase
	Phase string `json:"phase,omitempty"`

	// URL is the access URL for the Jupyter notebook
	URL string `json:"url,omitempty"`

	// Token is the Jupyter authentication token
	Token string `json:"token,omitempty"`

	// PodName is the name of the running pod
	PodName string `json:"podName,omitempty"`

	// StartTime is when the notebook started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// LastActivityTime is the last detected activity
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="User",type=string,JSONPath=`.spec.userID`
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// JupyterRuntime is the Schema for the jupyterruntimes API
type JupyterRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JupyterRuntimeSpec   `json:"spec,omitempty"`
	Status JupyterRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JupyterRuntimeList contains a list of JupyterRuntime
type JupyterRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JupyterRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JupyterRuntime{}, &JupyterRuntimeList{})
}
