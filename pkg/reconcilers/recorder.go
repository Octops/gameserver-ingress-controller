package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const (
	IngressKind = "Ingress"
	ServiceKind = "Service"
)

type EventRecorder struct {
	recorder record.EventRecorder
}

func NewEventRecorder(recorder record.EventRecorder) *EventRecorder {
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
