package reconcilers

import networkingv1 "k8s.io/api/networking/v1"

const (
	EventTypeNormal         string = "Normal"
	EventTypeWarning               = "Warning"
	ReasonReconcileFailed          = "Failed"
	ReasonReconciled               = "Created"
	ReasonReconcileCreating        = "Creating"

	NginxRewriteTargetAnnotation      = "nginx.ingress.kubernetes.io/rewrite-target"
	NginxRewriteTargetAnnotationValue = "/$2"
	NginxRewriteTargetPathFormat      = "/%s(/|$)(.*)"
)

var (
	defaultPathType = networkingv1.PathTypePrefix
)
