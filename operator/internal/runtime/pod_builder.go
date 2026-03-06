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

package runtime

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

const (
	// Container images
	SimulatorImage       = "blocksq/qiskit-runtime:latest"
	ResultCollectorImage = "blocksq/result-collector:latest"

	// Volume names
	CodeVolumeName   = "code"
	ResultVolumeName = "results"

	// Mount paths
	CodeMountPath   = "/code"
	ResultMountPath = "/results"

	// Environment variables
	QiskitMethodEnv     = "QISKIT_METHOD"
	MaxExecutionTimeEnv = "MAX_EXECUTION_TIME"
	S3BucketEnv         = "S3_BUCKET"
	JobIDEnv            = "JOB_ID"

	// Default values
	DefaultResultsBucket   = "quantum-results"
	DefaultResultSizeLimit = "1Gi"
)

// PodBuilder builds simulation pods from QuantumJob specs
type PodBuilder struct{}

// NewPodBuilder creates a new PodBuilder
func NewPodBuilder() *PodBuilder {
	return &PodBuilder{}
}

// BuildSimulationPod creates a simulation pod from QuantumJob
func (pb *PodBuilder) BuildSimulationPod(job *quantumv1alpha1.QuantumJob) (*corev1.Pod, error) {
	podName := fmt.Sprintf("qjob-%s-runner", job.Name)

	// Build main simulator container
	simulatorContainer, err := pb.buildSimulatorContainer(job)
	if err != nil {
		return nil, fmt.Errorf("failed to build simulator container: %w", err)
	}

	// Build result collector sidecar
	collectorContainer := pb.buildResultCollectorContainer(job)

	// Build volumes
	volumes := pb.buildVolumes(job)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"quantum-job":         job.Name,
				"quantum-job-user-id": job.Spec.UserID,
				"quantum-job-phase":   string(job.Status.Phase),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         job.APIVersion,
					Kind:               job.Kind,
					Name:               job.Name,
					UID:                job.UID,
					Controller:         &[]bool{true}[0],
					BlockOwnerDeletion: &[]bool{true}[0],
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers:            []corev1.Container{simulatorContainer, collectorContainer},
			Volumes:               volumes,
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: &[]int64{int64(job.Spec.Scheduling.Timeout)}[0],
		},
	}

	// Add node assignment if available
	if job.Status.AssignedNode != "" {
		pod.Spec.NodeName = job.Status.AssignedNode
	}

	return pod, nil
}

// buildSimulatorContainer creates the main quantum simulator container
func (pb *PodBuilder) buildSimulatorContainer(job *quantumv1alpha1.QuantumJob) (corev1.Container, error) {
	// Calculate resource requirements
	cpuQuantity, err := resource.ParseQuantity(job.Spec.Resources.CPU)
	if err != nil {
		return corev1.Container{}, fmt.Errorf("invalid CPU quantity: %w", err)
	}

	memoryQuantity, err := resource.ParseQuantity(job.Spec.Resources.Memory)
	if err != nil {
		return corev1.Container{}, fmt.Errorf("invalid memory quantity: %w", err)
	}

	// Set up environment variables
	envVars := []corev1.EnvVar{
		{
			Name:  QiskitMethodEnv,
			Value: string(job.Spec.Complexity.Method),
		},
		{
			Name:  MaxExecutionTimeEnv,
			Value: strconv.Itoa(int(job.Spec.Scheduling.Timeout)),
		},
	}

	container := corev1.Container{
		Name:    "simulator",
		Image:   SimulatorImage,
		Command: []string{"python", "/runner/execute.py"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      CodeVolumeName,
				MountPath: CodeMountPath,
				ReadOnly:  true,
			},
			{
				Name:      ResultVolumeName,
				MountPath: ResultMountPath,
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    cpuQuantity,
				corev1.ResourceMemory: memoryQuantity,
			},
			Requests: corev1.ResourceList{
				// Request 50% of limits for better scheduling
				corev1.ResourceCPU:    *resource.NewMilliQuantity(cpuQuantity.MilliValue()/2, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(memoryQuantity.Value()/2, resource.BinarySI),
			},
		},
		Env: envVars,
	}

	// Add GPU resource if specified
	if job.Spec.Resources.GPU != "" && job.Spec.Resources.GPU != "0" {
		gpuQuantity, err := resource.ParseQuantity(job.Spec.Resources.GPU)
		if err != nil {
			return container, fmt.Errorf("invalid GPU quantity: %w", err)
		}
		container.Resources.Limits["nvidia.com/gpu"] = gpuQuantity
		container.Resources.Requests["nvidia.com/gpu"] = gpuQuantity
	}

	return container, nil
}

// buildResultCollectorContainer creates the result collector sidecar
func (pb *PodBuilder) buildResultCollectorContainer(job *quantumv1alpha1.QuantumJob) corev1.Container {
	return corev1.Container{
		Name:  "result-collector",
		Image: ResultCollectorImage,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      ResultVolumeName,
				MountPath: ResultMountPath,
				ReadOnly:  true,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  S3BucketEnv,
				Value: DefaultResultsBucket,
			},
			{
				Name:  JobIDEnv,
				Value: job.Name,
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
	}
}

// buildVolumes creates the required volumes for the pod
func (pb *PodBuilder) buildVolumes(job *quantumv1alpha1.QuantumJob) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: CodeVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("qjob-%s-code", job.Name),
					},
				},
			},
		},
		{
			Name: ResultVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: func() *resource.Quantity {
						q := resource.MustParse(DefaultResultSizeLimit)
						return &q
					}(),
				},
			},
		},
	}
}

// BuildCodeConfigMap creates a ConfigMap containing the user code
func (pb *PodBuilder) BuildCodeConfigMap(job *quantumv1alpha1.QuantumJob) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("qjob-%s-code", job.Name),
			Namespace: job.Namespace,
			Labels: map[string]string{
				"quantum-job": job.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         job.APIVersion,
					Kind:               job.Kind,
					Name:               job.Name,
					UID:                job.UID,
					Controller:         &[]bool{true}[0],
					BlockOwnerDeletion: &[]bool{true}[0],
				},
			},
		},
		Data: map[string]string{
			"circuit.py": job.Spec.Circuit.Source,
			"language":   string(job.Spec.Circuit.Language),
			"version":    job.Spec.Circuit.Version,
		},
	}
}

// SetPodNodeAffinity adds node pool affinity to the pod
func (pb *PodBuilder) SetPodNodeAffinity(pod *corev1.Pod, nodePool quantumv1alpha1.NodePool) {
	if nodePool == quantumv1alpha1.NodePoolAuto {
		return // No specific affinity for auto
	}

	affinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "quantum.blocksq.io/pool",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{string(nodePool)},
							},
						},
					},
				},
			},
		},
	}

	pod.Spec.Affinity = affinity
}
