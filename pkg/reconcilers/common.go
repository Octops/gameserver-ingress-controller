package reconcilers

import networkingv1 "k8s.io/api/networking/v1"

const (
	EventTypeNormal         string = "Normal"
	EventTypeWarning               = "Warning"
	ReasonReconcileFailed          = "Failed"
	ReasonReconciled               = "Created"
	ReasonReconcileCreating        = "Creating"
)

var (
	defaultPathType = networkingv1.PathTypePrefix
)
