package gameserver

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

type IngressRoutingMode string

const (
	IngressRoutingModeDomain IngressRoutingMode = "domain"
	IngressRoutingModePath   IngressRoutingMode = "path"

	OctopsAnnotationIngressMode            = "octops.io/gameserver-ingress-mode"
	OctopsAnnotationIngressDomain          = "octops.io/gameserver-ingress-domain"
	OctopsAnnotationIngressFQDN            = "octops.io/gameserver-ingress-fqdn"
	OctopsAnnotationTerminateTLS           = "octops.io/terminate-tls"
	OctopsAnnotationsTLSSecretName         = "octops.io/tls-secret-name"
	OctopsAnnotationIssuerName             = "octops.io/issuer-tls-name"
	OctopsAnnotationCustomPrefix           = "octops-"
	OctopsAnnotationCustomServicePrefix    = "octops.service-"
	OctopsAnnotationGameServerIngressReady = "octops.io/ingress-ready"
	OctopsAnnotationIngressClassName       = "octops.io/ingress-class-name"
	OctopsAnnotationIngressClassNameLegacy = "octops-kubernetes.io/ingress.class"

	CertManagerAnnotationIssuer = "cert-manager.io/cluster-issuer"
	AgonesGameServerNameLabel   = "agones.dev/gameserver"

	ErrGameServerAnnotationMissing = "gameserver %s/%s is missing annotation %s"
	ErrGameServerAnnotationEmpty   = "gameserver %s/%s has annotation %s but it is empty"
	ErrIngressRoutingModeEmpty     = "ingress routing mode %s requires the annotation %s to be set on gameserver %s/%s"
)

func (m IngressRoutingMode) String() string {
	return string(m)
}

func FromObject(obj interface{}) *agonesv1.GameServer {
	if gs, ok := obj.(*agonesv1.GameServer); ok {
		return gs
	}

	return &agonesv1.GameServer{}
}

func GetGameServerPort(gs *agonesv1.GameServer) agonesv1.GameServerStatusPort {
	if len(gs.Status.Ports) > 0 {
		return gs.Status.Ports[0]
	}

	return agonesv1.GameServerStatusPort{}
}

func GetGameServerContainerPort(gs *agonesv1.GameServer) int32 {
	if len(gs.Spec.Ports) > 0 {
		return gs.Spec.Ports[0].ContainerPort
	}

	return 0
}

func HasAnnotation(gs *agonesv1.GameServer, annotation string) (string, bool) {
	if value, ok := gs.Annotations[annotation]; ok {
		return value, true
	}

	return "", false
}

func IsShutdown(gs *agonesv1.GameServer) bool {
	if gs == nil {
		return false
	}

	return gs.Status.State == agonesv1.GameServerStateShutdown
}

func MustReconcile(gs *agonesv1.GameServer) bool {
	if gs == nil {
		return false
	}

	switch gs.Status.State {
	case agonesv1.GameServerStateScheduled,
		agonesv1.GameServerStateRequestReady,
		agonesv1.GameServerStateReady:
		return true
	}

	return false
}

func GetIngressRoutingMode(gs *agonesv1.GameServer) IngressRoutingMode {
	if mode, ok := HasAnnotation(gs, OctopsAnnotationIngressMode); ok {
		return IngressRoutingMode(mode)
	}

	return IngressRoutingModeDomain
}

func GetTLSCertIssuer(gs *agonesv1.GameServer) string {
	if name, ok := HasAnnotation(gs, OctopsAnnotationIssuerName); ok {
		return name
	}

	return ""
}

func GetIngressClassName(gs *agonesv1.GameServer) string {
	if className, ok := HasAnnotation(gs, OctopsAnnotationIngressClassName); ok {
		return className
	}

	if className, ok := HasAnnotation(gs, OctopsAnnotationIngressClassNameLegacy); ok {
		return className
	}

	return ""
}
