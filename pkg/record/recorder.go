package record

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	IngressKind = "Ingress"
	ServiceKind = "Service"

	EventTypeNormal         string = "Normal"
	EventTypeWarning               = "Warning"
	ReasonReconcileFailed          = "Failed"
	ReasonReconciled               = "Created"
	ReasonReconcileCreating        = "Creating"
)

type Recorder interface {
	Event(object runtime.Object, eventtype string, reason string, message string)
}

type EventRecorder struct {
	recorder Recorder
}

func NewEventRecorder(recorder Recorder) *EventRecorder {
	return &EventRecorder{recorder: recorder}
}

func (r *EventRecorder) RecordFailed(gs *agonesv1.GameServer, kind string, err error) {
	r.recordEvent(gs, EventTypeWarning, ReasonReconcileFailed, fmt.Sprintf("Failed to create %s for gameserver %s/%s: %s", kind, gs.Namespace, gs.Name, err))
}

func (r *EventRecorder) RecordSuccess(gs *agonesv1.GameServer, kind string) {
	r.recordEvent(gs, EventTypeNormal, ReasonReconciled, fmt.Sprintf("%s created for gameserver %s/%s", kind, gs.Namespace, gs.Name))
}

func (r *EventRecorder) RecordCreating(gs *agonesv1.GameServer, kind string) {
	r.recordEvent(gs, EventTypeNormal, ReasonReconcileCreating, fmt.Sprintf("Creating %s for gameserver %s/%s", kind, gs.Namespace, gs.Name))
}

func (r *EventRecorder) recordEvent(object runtime.Object, eventtype, reason, message string) {
	r.recorder.Event(object, eventtype, reason, message)
}
