package gameserver

import agonesv1 "agones.dev/agones/pkg/apis/agones/v1"

const (
	OctopsAnnotationIngressMode   = "octops.io/gameserver-ingress-mode"
	OctopsAnnotationIngressDomain = "octops.io/gameserver-ingress-domain"
	OctopsAnnotationIngressFQDN   = "octops.io/gameserver-ingress-fqdn"
	OctopsAnnotationTerminateTLS  = "octops.io/terminate-tls"
	OctopsAnnotationIssuerName    = "octops.io/issuer-tls-name"

	CertManagerAnnotationIssuer = "cert-manager.io/issuer"
	AgonesGameServerNameLabel   = "agones.dev/gameserver"
)

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

func IsReady(gs *agonesv1.GameServer) bool {
	if gs == nil {
		return false
	}

	return gs.Status.State == agonesv1.GameServerStateReady
}
