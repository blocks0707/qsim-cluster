package runtime

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodBuilder_BuildSimulationPod(t *testing.T) {
	tests := []struct {
		name    string
		job     *quantumv1alpha1.QuantumJob
		want    func(t *testing.T, pod *corev1.Pod)
		wantErr bool
	}{
		{
			name: "basic pod creation - single container only",
			job: &quantumv1alpha1.QuantumJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
					UID:       types.UID("test-uid"),
				},
				Spec: quantumv1alpha1.QuantumJobSpec{
					UserID: "test-user",
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "2",
						Memory: "4Gi",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Timeout: 300,
					},
					Complexity: &quantumv1alpha1.ComplexitySpec{
						Method: quantumv1alpha1.SimulationMethodStatevector,
					},
					Circuit: quantumv1alpha1.CircuitSpec{
						Source:   "print('hello world')",
						Language: quantumv1alpha1.CodeLanguagePython,
						Version:  "3.8",
					},
				},
			},
			want: func(t *testing.T, pod *corev1.Pod) {
				assert.Equal(t, "qjob-test-job-runner", pod.Name)
				assert.Equal(t, "default", pod.Namespace)
				assert.Equal(t, "test-job", pod.Labels["quantum-job"])
				assert.Equal(t, "test-user", pod.Labels["quantum-job-user-id"])

				// Must have exactly 1 container (no result-collector sidecar)
				assert.Len(t, pod.Spec.Containers, 1)

				simulator := pod.Spec.Containers[0]
				assert.Equal(t, "simulator", simulator.Name)
				assert.Equal(t, SimulatorImage, simulator.Image)
				assert.Equal(t, []string{"python", "/app/execute.py"}, simulator.Command)

				// Check resource limits
				assert.Equal(t, resource.MustParse("2"), simulator.Resources.Limits[corev1.ResourceCPU])
				assert.Equal(t, resource.MustParse("4Gi"), simulator.Resources.Limits[corev1.ResourceMemory])

				// Check volume mounts
				assert.Len(t, simulator.VolumeMounts, 2)
				codeMount := findVolumeMount(simulator.VolumeMounts, CodeVolumeName)
				require.NotNil(t, codeMount)
				assert.Equal(t, CodeMountPath, codeMount.MountPath)
				assert.True(t, codeMount.ReadOnly)

				resultMount := findVolumeMount(simulator.VolumeMounts, ResultVolumeName)
				require.NotNil(t, resultMount)
				assert.Equal(t, ResultMountPath, resultMount.MountPath)
				assert.False(t, resultMount.ReadOnly)

				// Check env vars
				assert.Len(t, simulator.Env, 2)

				// Check volumes
				assert.Len(t, pod.Spec.Volumes, 2)
				codeVolume := findVolume(pod.Spec.Volumes, CodeVolumeName)
				require.NotNil(t, codeVolume)
				assert.Equal(t, "qjob-test-job-code", codeVolume.ConfigMap.Name)

				// Check restart policy and deadline
				assert.Equal(t, corev1.RestartPolicyNever, pod.Spec.RestartPolicy)
				assert.Equal(t, int64(300), *pod.Spec.ActiveDeadlineSeconds)
			},
		},
		{
			name: "pod with GPU resources",
			job: &quantumv1alpha1.QuantumJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gpu-job",
					Namespace: "default",
					UID:       types.UID("gpu-uid"),
				},
				Spec: quantumv1alpha1.QuantumJobSpec{
					UserID: "gpu-user",
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "4",
						Memory: "8Gi",
						GPU:    "1",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Timeout: 600,
					},
					Complexity: &quantumv1alpha1.ComplexitySpec{
						Method: quantumv1alpha1.SimulationMethodStatevector,
					},
					Circuit: quantumv1alpha1.CircuitSpec{
						Source:   "print('gpu test')",
						Language: quantumv1alpha1.CodeLanguagePython,
						Version:  "3.9",
					},
				},
			},
			want: func(t *testing.T, pod *corev1.Pod) {
				assert.Len(t, pod.Spec.Containers, 1)
				simulator := pod.Spec.Containers[0]
				gpuLimit := simulator.Resources.Limits["nvidia.com/gpu"]
				assert.Equal(t, resource.MustParse("1"), gpuLimit)
			},
		},
		{
			name: "pod with assigned node",
			job: &quantumv1alpha1.QuantumJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "node-job",
					Namespace: "default",
					UID:       types.UID("node-uid"),
				},
				Spec: quantumv1alpha1.QuantumJobSpec{
					UserID: "node-user",
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "1",
						Memory: "2Gi",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Timeout: 180,
					},
					Complexity: &quantumv1alpha1.ComplexitySpec{
						Method: quantumv1alpha1.SimulationMethodStatevector,
					},
					Circuit: quantumv1alpha1.CircuitSpec{
						Source:   "print('node test')",
						Language: quantumv1alpha1.CodeLanguagePython,
						Version:  "3.8",
					},
				},
				Status: quantumv1alpha1.QuantumJobStatus{
					AssignedNode: "node-1",
				},
			},
			want: func(t *testing.T, pod *corev1.Pod) {
				assert.Equal(t, "node-1", pod.Spec.NodeName)
			},
		},
		{
			name: "invalid CPU resource returns error",
			job: &quantumv1alpha1.QuantumJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-job",
					Namespace: "default",
					UID:       types.UID("invalid-uid"),
				},
				Spec: quantumv1alpha1.QuantumJobSpec{
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "invalid-cpu",
						Memory: "4Gi",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Timeout: 300,
					},
					Complexity: &quantumv1alpha1.ComplexitySpec{
						Method: quantumv1alpha1.SimulationMethodStatevector,
					},
					Circuit: quantumv1alpha1.CircuitSpec{
						Source:   "print('test')",
						Language: quantumv1alpha1.CodeLanguagePython,
						Version:  "3.8",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPodBuilder()
			pod, err := pb.BuildSimulationPod(tt.job)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pod)

			if tt.want != nil {
				tt.want(t, pod)
			}
		})
	}
}

func TestPodBuilder_BuildCodeConfigMap(t *testing.T) {
	job := &quantumv1alpha1.QuantumJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-job",
			Namespace:  "default",
			UID:        types.UID("test-uid"),
		},
		Spec: quantumv1alpha1.QuantumJobSpec{
			Circuit: quantumv1alpha1.CircuitSpec{
				Source:   "from qiskit import QuantumCircuit\nqc = QuantumCircuit(2)",
				Language: quantumv1alpha1.CodeLanguagePython,
				Version:  "3.9",
			},
		},
	}

	pb := NewPodBuilder()
	cm := pb.BuildCodeConfigMap(job)

	assert.Equal(t, "qjob-test-job-code", cm.Name)
	assert.Equal(t, "default", cm.Namespace)
	assert.Equal(t, "test-job", cm.Labels["quantum-job"])
	assert.Equal(t, job.Spec.Circuit.Source, cm.Data["circuit.py"])

	require.Len(t, cm.OwnerReferences, 1)
	assert.True(t, *cm.OwnerReferences[0].Controller)
}

func TestPodBuilder_SetPodNodeAffinity(t *testing.T) {
	tests := []struct {
		name     string
		nodePool quantumv1alpha1.NodePool
		wantNil  bool
	}{
		{"auto pool - no affinity", quantumv1alpha1.NodePoolAuto, true},
		{"cpu pool - adds affinity", quantumv1alpha1.NodePoolCPU, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{Spec: corev1.PodSpec{}}
			pb := NewPodBuilder()
			pb.SetPodNodeAffinity(pod, tt.nodePool)

			if tt.wantNil {
				assert.Nil(t, pod.Spec.Affinity)
			} else {
				require.NotNil(t, pod.Spec.Affinity)
				require.NotNil(t, pod.Spec.Affinity.NodeAffinity)
			}
		})
	}
}

// Helpers
func findVolumeMount(mounts []corev1.VolumeMount, name string) *corev1.VolumeMount {
	for _, m := range mounts {
		if m.Name == name {
			return &m
		}
	}
	return nil
}

func findVolume(volumes []corev1.Volume, name string) *corev1.Volume {
	for _, v := range volumes {
		if v.Name == name {
			return &v
		}
	}
	return nil
}
