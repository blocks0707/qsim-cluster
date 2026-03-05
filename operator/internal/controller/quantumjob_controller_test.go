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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

var _ = Describe("QuantumJob Controller", func() {
	Context("When creating a QuantumJob", func() {
		const (
			jobName      = "test-job"
			jobNamespace = "default"
			timeout      = time.Second * 10
			duration     = time.Second * 10
			interval     = time.Millisecond * 250
		)

		It("Should create job and transition through phases", func() {
			By("Creating a new QuantumJob")
			job := &quantumv1alpha1.QuantumJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: jobNamespace,
				},
				Spec: quantumv1alpha1.QuantumJobSpec{
					UserID: "test-user",
					Circuit: quantumv1alpha1.CircuitSpec{
						Source:   "from qiskit import QuantumCircuit\nqc = QuantumCircuit(2)\nqc.h(0)\nqc.cx(0,1)\nprint('Hello Quantum!')",
						Language: quantumv1alpha1.CodeLanguagePython,
						Version:  "3.11",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Priority: quantumv1alpha1.JobPriorityNormal,
						NodePool: quantumv1alpha1.NodePoolAuto,
						Timeout:  300,
						RetryPolicy: quantumv1alpha1.RetryPolicy{
							MaxRetries:     2,
							BackoffSeconds: 30,
						},
					},
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "2",
						Memory: "4Gi",
						GPU:    "0",
					},
				},
			}

			Expect(k8sClient.Create(context.Background(), job)).Should(Succeed())

			jobLookupKey := types.NamespacedName{Name: jobName, Namespace: jobNamespace}
			createdJob := &quantumv1alpha1.QuantumJob{}

			By("Checking the job moves to Pending phase")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), jobLookupKey, createdJob)
				if err != nil {
					return false
				}
				return createdJob.Status.Phase == quantumv1alpha1.JobPhasePending
			}, timeout, interval).Should(BeTrue())

			By("Checking the job moves to Analyzing phase")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), jobLookupKey, createdJob)
				if err != nil {
					return false
				}
				return createdJob.Status.Phase == quantumv1alpha1.JobPhaseAnalyzing
			}, timeout, interval).Should(BeTrue())

			By("Checking the job gets complexity metadata")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), jobLookupKey, createdJob)
				if err != nil {
					return false
				}
				return createdJob.Spec.Complexity != nil
			}, timeout, interval).Should(BeTrue())

			By("Checking the job moves to Scheduling phase")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), jobLookupKey, createdJob)
				if err != nil {
					return false
				}
				return createdJob.Status.Phase == quantumv1alpha1.JobPhaseScheduling
			}, timeout, interval).Should(BeTrue())

			By("Cleaning up the job")
			Expect(k8sClient.Delete(context.Background(), job)).Should(Succeed())
		})
	})

	Context("When validating job specifications", func() {
		It("Should reject jobs with invalid specifications", func() {
			reconciler := &QuantumJobReconciler{}
			
			// Test missing UserID
			invalidJob := &quantumv1alpha1.QuantumJob{
				Spec: quantumv1alpha1.QuantumJobSpec{
					Circuit: quantumv1alpha1.CircuitSpec{
						Source: "test code",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Timeout: 300,
					},
				},
			}
			
			err := reconciler.validateJobSpec(invalidJob)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("userID is required"))

			// Test missing circuit source
			invalidJob.Spec.UserID = "test-user"
			invalidJob.Spec.Circuit.Source = ""
			err = reconciler.validateJobSpec(invalidJob)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("circuit source code is required"))

			// Test invalid timeout
			invalidJob.Spec.Circuit.Source = "test code"
			invalidJob.Spec.Scheduling.Timeout = 0
			err = reconciler.validateJobSpec(invalidJob)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("timeout must be positive"))

			// Test valid job
			invalidJob.Spec.Scheduling.Timeout = 300
			err = reconciler.validateJobSpec(invalidJob)
			Expect(err).Should(Succeed())
		})
	})

	Context("When handling job events", func() {
		It("Should add events and maintain event limit", func() {
			reconciler := &QuantumJobReconciler{}
			job := &quantumv1alpha1.QuantumJob{
				Status: quantumv1alpha1.QuantumJobStatus{},
			}

			// Add multiple events
			for i := 0; i < 15; i++ {
				reconciler.addJobEvent(job, "Normal", "TestEvent", "Test message")
			}

			// Should only keep last 10 events
			Expect(len(job.Status.Events)).Should(Equal(10))

			// Check that events have timestamps
			for _, event := range job.Status.Events {
				Expect(event.Timestamp).ShouldNot(BeZero())
				Expect(event.Type).Should(Equal("Normal"))
				Expect(event.Reason).Should(Equal("TestEvent"))
				Expect(event.Message).Should(Equal("Test message"))
			}
		})
	})

	Context("When managing job conditions", func() {
		It("Should set and update conditions correctly", func() {
			reconciler := &QuantumJobReconciler{}
			var conditions []metav1.Condition

			// Add first condition
			now := metav1.NewTime(time.Now())
			condition1 := metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "JobReady",
				Message:            "Job is ready",
			}
			reconciler.setJobCondition(&conditions, condition1)
			
			Expect(len(conditions)).Should(Equal(1))
			Expect(conditions[0].Type).Should(Equal("Ready"))
			Expect(conditions[0].Status).Should(Equal(metav1.ConditionTrue))

			// Update existing condition
			condition2 := metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: now,
				Reason:             "JobNotReady",
				Message:            "Job is not ready",
			}
			reconciler.setJobCondition(&conditions, condition2)
			
			// Should still have only one condition, but updated
			Expect(len(conditions)).Should(Equal(1))
			Expect(conditions[0].Type).Should(Equal("Ready"))
			Expect(conditions[0].Status).Should(Equal(metav1.ConditionFalse))
			Expect(conditions[0].Reason).Should(Equal("JobNotReady"))

			// Add different condition type
			condition3 := metav1.Condition{
				Type:               "Scheduled",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "NodeAssigned",
				Message:            "Node assigned successfully",
			}
			reconciler.setJobCondition(&conditions, condition3)
			
			// Should now have two conditions
			Expect(len(conditions)).Should(Equal(2))
		})
	})
})

var _ = Describe("QuantumJob Integration Tests", func() {
	Context("When running a complete job lifecycle", func() {
		It("Should handle job with node profiles available", func() {
			const (
				jobName      = "integration-job"
				nodeName     = "test-node"
				jobNamespace = "default"
			)

			By("Creating a test node profile")
			nodeProfile := &quantumv1alpha1.QuantumNodeProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: jobNamespace,
				},
				Spec: quantumv1alpha1.QuantumNodeProfileSpec{
					Pool: quantumv1alpha1.NodePoolCPU,
					CPU: quantumv1alpha1.CPUCapabilities{
						Cores:        4,
						Architecture: quantumv1alpha1.CPUArchitectureX86_64,
					},
					Memory: quantumv1alpha1.MemoryCapabilities{
						TotalGB: 8,
					},
					GPU: quantumv1alpha1.GPUCapabilities{
						Available: false,
					},
					SimulatorConfig: quantumv1alpha1.SimulatorConfig{
						MaxConcurrentJobs: 3,
						SupportedMethods: []quantumv1alpha1.SimulationMethod{
							quantumv1alpha1.SimulationMethodStatevector,
							quantumv1alpha1.SimulationMethodStabilizer,
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), nodeProfile)).Should(Succeed())

			By("Updating node profile status to ready")
			nodeProfile.Status = quantumv1alpha1.QuantumNodeProfileStatus{
				Ready: true,
				CurrentLoad: &quantumv1alpha1.LoadStatus{
					CPUUsagePercent:    20,
					MemoryUsagePercent: 30,
					ActiveJobs:         0,
				},
				LastUpdated: &metav1.Time{Time: time.Now()},
			}
			Expect(k8sClient.Status().Update(context.Background(), nodeProfile)).Should(Succeed())

			By("Creating a quantum job")
			job := &quantumv1alpha1.QuantumJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: jobNamespace,
				},
				Spec: quantumv1alpha1.QuantumJobSpec{
					UserID: "test-user",
					Circuit: quantumv1alpha1.CircuitSpec{
						Source:   "from qiskit import QuantumCircuit\nqc = QuantumCircuit(2,2)\nqc.h(0)\nqc.cx(0,1)\nqc.measure_all()",
						Language: quantumv1alpha1.CodeLanguagePython,
						Version:  "3.11",
					},
					Scheduling: quantumv1alpha1.SchedulingSpec{
						Priority: quantumv1alpha1.JobPriorityNormal,
						NodePool: quantumv1alpha1.NodePoolCPU,
						Timeout:  300,
					},
					Resources: quantumv1alpha1.ResourceSpec{
						CPU:    "2",
						Memory: "4Gi",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), job)).Should(Succeed())

			jobLookupKey := types.NamespacedName{Name: jobName, Namespace: jobNamespace}
			createdJob := &quantumv1alpha1.QuantumJob{}

			By("Waiting for job to be scheduled to the node")
			Eventually(func() string {
				err := k8sClient.Get(context.Background(), jobLookupKey, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.AssignedNode
			}, time.Second*15, time.Millisecond*250).Should(Equal(nodeName))

			By("Checking that job reaches Running phase")
			Eventually(func() quantumv1alpha1.JobPhase {
				err := k8sClient.Get(context.Background(), jobLookupKey, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.Phase
			}, time.Second*15, time.Millisecond*250).Should(Equal(quantumv1alpha1.JobPhaseRunning))

			By("Verifying job has complexity analysis")
			Expect(createdJob.Spec.Complexity).ShouldNot(BeNil())
			Expect(createdJob.Spec.Complexity.Qubits).Should(BeNumerically(">", 0))

			By("Verifying job has correct scheduling conditions")
			var scheduledCondition *metav1.Condition
			for _, condition := range createdJob.Status.Conditions {
				if condition.Type == "Scheduled" {
					scheduledCondition = &condition
					break
				}
			}
			Expect(scheduledCondition).ShouldNot(BeNil())
			Expect(scheduledCondition.Status).Should(Equal(metav1.ConditionTrue))
			Expect(scheduledCondition.Reason).Should(Equal("NodeAssigned"))

			By("Cleaning up")
			Expect(k8sClient.Delete(context.Background(), job)).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), nodeProfile)).Should(Succeed())
		})
	})
})