/*
Copyright 2024.
Licensed under the Apache License, Version 2.0.
*/

package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	quantumv1alpha1 "github.com/mungch0120/qsim-cluster/operator/api/v1alpha1"
)

// JupyterRuntimeReconciler reconciles a JupyterRuntime object
type JupyterRuntimeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=jupyterruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=quantum.blocksq.io,resources=jupyterruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;delete

func (r *JupyterRuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var jr quantumv1alpha1.JupyterRuntime
	if err := r.Get(ctx, req.NamespacedName, &jr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling JupyterRuntime", "name", jr.Name, "phase", jr.Status.Phase)

	switch jr.Status.Phase {
	case "":
		return r.handleNew(ctx, &jr)
	case quantumv1alpha1.JupyterPhasePending:
		return r.handlePending(ctx, &jr)
	case quantumv1alpha1.JupyterPhaseProvisioning:
		return r.handleProvisioning(ctx, &jr)
	case quantumv1alpha1.JupyterPhaseRunning:
		return r.handleRunning(ctx, &jr)
	case quantumv1alpha1.JupyterPhaseStopped, quantumv1alpha1.JupyterPhaseFailed:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}
}

func (r *JupyterRuntimeReconciler) handleNew(ctx context.Context, jr *quantumv1alpha1.JupyterRuntime) (ctrl.Result, error) {
	// Generate token
	token := generateToken()
	jr.Status.Phase = quantumv1alpha1.JupyterPhasePending
	jr.Status.Token = token

	if err := r.Status().Update(ctx, jr); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *JupyterRuntimeReconciler) handlePending(ctx context.Context, jr *quantumv1alpha1.JupyterRuntime) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create PVC for notebook storage
	pvc := r.buildPVC(jr)
	if err := r.Create(ctx, pvc); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create PVC")
		return ctrl.Result{}, err
	}

	// Create the Jupyter pod
	pod := r.buildPod(jr)
	if err := r.Create(ctx, pod); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create Jupyter pod")
		return ctrl.Result{}, err
	}

	// Create service
	svc := r.buildService(jr)
	if err := r.Create(ctx, svc); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create Jupyter service")
		return ctrl.Result{}, err
	}

	jr.Status.Phase = quantumv1alpha1.JupyterPhaseProvisioning
	jr.Status.PodName = pod.Name

	if err := r.Status().Update(ctx, jr); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *JupyterRuntimeReconciler) handleProvisioning(ctx context.Context, jr *quantumv1alpha1.JupyterRuntime) (ctrl.Result, error) {
	podName := fmt.Sprintf("jupyter-%s", jr.Name)
	var pod corev1.Pod
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: jr.Namespace}, &pod); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	switch pod.Status.Phase {
	case corev1.PodRunning:
		now := metav1.Now()
		jr.Status.Phase = quantumv1alpha1.JupyterPhaseRunning
		jr.Status.StartTime = &now
		jr.Status.LastActivityTime = &now
		// Service URL (cluster-internal)
		jr.Status.URL = fmt.Sprintf("http://jupyter-%s.%s.svc:8888", jr.Name, jr.Namespace)

		if err := r.Status().Update(ctx, jr); err != nil {
			if errors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	case corev1.PodFailed:
		jr.Status.Phase = quantumv1alpha1.JupyterPhaseFailed
		r.Status().Update(ctx, jr)
		return ctrl.Result{}, nil

	default:
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
}

func (r *JupyterRuntimeReconciler) handleRunning(ctx context.Context, jr *quantumv1alpha1.JupyterRuntime) (ctrl.Result, error) {
	// Check pod still exists
	podName := fmt.Sprintf("jupyter-%s", jr.Name)
	var pod corev1.Pod
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: jr.Namespace}, &pod); err != nil {
		if errors.IsNotFound(err) {
			jr.Status.Phase = quantumv1alpha1.JupyterPhaseStopped
			r.Status().Update(ctx, jr)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		jr.Status.Phase = quantumv1alpha1.JupyterPhaseStopped
		r.Status().Update(ctx, jr)
		return ctrl.Result{}, nil
	}

	// Check idle timeout
	timeout := time.Duration(jr.Spec.Timeout) * time.Second
	if jr.Status.LastActivityTime != nil {
		idleDuration := time.Since(jr.Status.LastActivityTime.Time)
		if idleDuration > timeout {
			log.FromContext(ctx).Info("Jupyter session idle timeout, stopping",
				"name", jr.Name, "idle", idleDuration)
			// Delete pod
			r.Delete(ctx, &pod)
			jr.Status.Phase = quantumv1alpha1.JupyterPhaseStopped
			r.Status().Update(ctx, jr)
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

func (r *JupyterRuntimeReconciler) buildPod(jr *quantumv1alpha1.JupyterRuntime) *corev1.Pod {
	image := jr.Spec.Image
	if image == "" {
		image = "jupyter/scipy-notebook:latest"
	}

	cpu := jr.Spec.Resources.CPU
	if cpu == "" {
		cpu = "2"
	}
	memory := jr.Spec.Resources.Memory
	if memory == "" {
		memory = "4Gi"
	}

	// Build startup script for additional packages
	startCmd := "start-notebook.sh"
	args := []string{
		fmt.Sprintf("--NotebookApp.token='%s'", jr.Status.Token),
		"--NotebookApp.allow_origin='*'",
		"--NotebookApp.ip='0.0.0.0'",
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("jupyter-%s", jr.Name),
			Namespace: jr.Namespace,
			Labels: map[string]string{
				"app":                  "jupyter",
				"jupyter-runtime":      jr.Name,
				"quantum.blocksq.io/user": jr.Spec.UserID,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "jupyter",
					Image:   image,
					Command: []string{startCmd},
					Args:    args,
					Ports: []corev1.ContainerPort{
						{ContainerPort: 8888, Name: "http"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpu),
							corev1.ResourceMemory: resource.MustParse(memory),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpu),
							corev1.ResourceMemory: resource.MustParse(memory),
						},
					},
					Env: []corev1.EnvVar{
						{Name: "QSIM_API_ENDPOINT", Value: jr.Spec.QSimEndpoint},
						{Name: "JUPYTER_ENABLE_LAB", Value: "yes"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "notebooks", MountPath: "/home/jovyan/work"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/api",
								Port: intstr.FromInt(8888),
							},
						},
						InitialDelaySeconds: 10,
						PeriodSeconds:       5,
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "notebooks",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("jupyter-%s-notebooks", jr.Name),
						},
					},
				},
			},
		},
	}

	// Add init container for additional packages
	if len(jr.Spec.Packages) > 0 {
		installCmd := "pip install"
		for _, pkg := range jr.Spec.Packages {
			installCmd += " " + pkg
		}
		pod.Spec.InitContainers = []corev1.Container{
			{
				Name:    "install-packages",
				Image:   image,
				Command: []string{"sh", "-c", installCmd},
			},
		}
	}

	return pod
}

func (r *JupyterRuntimeReconciler) buildService(jr *quantumv1alpha1.JupyterRuntime) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("jupyter-%s", jr.Name),
			Namespace: jr.Namespace,
			Labels: map[string]string{
				"app":             "jupyter",
				"jupyter-runtime": jr.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"jupyter-runtime": jr.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8888,
					TargetPort: intstr.FromInt(8888),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func (r *JupyterRuntimeReconciler) buildPVC(jr *quantumv1alpha1.JupyterRuntime) *corev1.PersistentVolumeClaim {
	storageSize := jr.Spec.Resources.Storage
	if storageSize == "" {
		storageSize = "10Gi"
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("jupyter-%s-notebooks", jr.Name),
			Namespace: jr.Namespace,
			Labels: map[string]string{
				"app":             "jupyter",
				"jupyter-runtime": jr.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}
}

func generateToken() string {
	b := make([]byte, 24)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SetupWithManager sets up the controller with the Manager
func (r *JupyterRuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&quantumv1alpha1.JupyterRuntime{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
