package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// JobStatusUpdate represents a status change from K8s CR
type JobStatusUpdate struct {
	JobID          string
	Phase          string
	AssignedNode   string
	AssignedPool   string
	StartTime      *time.Time
	CompletionTime *time.Time
	ExecutionTime  *int32
	ErrorMessage   string
}

// StatusSyncCallback is called when a QuantumJob CR status changes
type StatusSyncCallback func(update *JobStatusUpdate)

// Syncer watches QuantumJob CRs and syncs status changes
type Syncer struct {
	client   *Client
	logger   *zap.Logger
	callback StatusSyncCallback
	stopCh   chan struct{}
}

// NewSyncer creates a new K8s→DB status syncer
func NewSyncer(client *Client, callback StatusSyncCallback, logger *zap.Logger) *Syncer {
	return &Syncer{
		client:   client,
		logger:   logger,
		callback: callback,
		stopCh:   make(chan struct{}),
	}
}

// Start begins watching QuantumJob CRs
func (s *Syncer) Start(ctx context.Context) error {
	gvr := schema.GroupVersionResource{
		Group:    "quantum.blocksq.io",
		Version:  "v1alpha1",
		Resource: "quantumjobs",
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		s.client.dynamicClient,
		30*time.Second, // resync period
		"quantum-jobs", // namespace
		nil,
	)

	informer := factory.ForResource(gvr).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.handleUpdate(oldObj, newObj)
		},
	})

	s.logger.Info("Starting QuantumJob status syncer")

	go informer.Run(s.stopCh)

	// Wait for cache sync
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return fmt.Errorf("failed to sync informer cache")
	}

	s.logger.Info("QuantumJob syncer cache synced")
	return nil
}

// Stop stops the syncer
func (s *Syncer) Stop() {
	close(s.stopCh)
}

func (s *Syncer) handleUpdate(oldObj, newObj interface{}) {
	newUn, ok := newObj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	oldUn, ok := oldObj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	// Extract phases
	newPhase, _, _ := unstructured.NestedString(newUn.Object, "status", "phase")
	oldPhase, _, _ := unstructured.NestedString(oldUn.Object, "status", "phase")

	// Only process if phase changed
	if newPhase == oldPhase {
		return
	}

	// Extract job ID from CR name (format: qjob-{uuid})
	name := newUn.GetName()
	jobID := strings.TrimPrefix(name, "qjob-")
	if jobID == name {
		// Not a qjob- prefixed name, skip
		return
	}

	s.logger.Info("QuantumJob phase changed",
		zap.String("job_id", jobID),
		zap.String("old_phase", oldPhase),
		zap.String("new_phase", newPhase),
	)

	update := &JobStatusUpdate{
		JobID: jobID,
		Phase: newPhase,
	}

	// Extract additional status fields
	status, found, _ := unstructured.NestedMap(newUn.Object, "status")
	if found {
		if node, ok, _ := unstructured.NestedString(status, "assignedNode"); ok {
			update.AssignedNode = node
		}
		if pool, ok, _ := unstructured.NestedString(status, "assignedPool"); ok {
			update.AssignedPool = pool
		}
		if errMsg, ok, _ := unstructured.NestedString(status, "errorMessage"); ok {
			update.ErrorMessage = errMsg
		}
		if startStr, ok, _ := unstructured.NestedString(status, "startTime"); ok {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				update.StartTime = &t
			}
		}
		if compStr, ok, _ := unstructured.NestedString(status, "completionTime"); ok {
			if t, err := time.Parse(time.RFC3339, compStr); err == nil {
				update.CompletionTime = &t
			}
		}
		if execTime, ok, _ := unstructured.NestedInt64(status, "executionTimeSec"); ok {
			execMs := execTime * 1000
			update.ExecutionTime = new(int32)
			*update.ExecutionTime = int32(execMs)
		}
	}

	s.callback(update)
}

// MapPhaseToDBStatus maps K8s CR phase to DB status string
func MapPhaseToDBStatus(phase string) string {
	switch phase {
	case "Pending":
		return "pending"
	case "Analyzing":
		return "analyzing"
	case "Scheduling":
		return "scheduling"
	case "Running":
		return "running"
	case "Succeeded":
		return "completed"
	case "Failed":
		return "failed"
	case "Cancelled":
		return "cancelled"
	default:
		return "unknown"
	}
}
