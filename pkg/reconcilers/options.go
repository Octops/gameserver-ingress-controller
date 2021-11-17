package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type IngressOption func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error

func WithTLS(mode IngressRoutingMode) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		tlsForDomain := func(host, domain string) (fqdn, secretName string) {
			return fmt.Sprintf("%s.%s", host, domain), fmt.Sprintf("%s-tls", host)
		}

		fqdn, secret := tlsForDomain(gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain])
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts: []string{
					fqdn,
				},
				SecretName: secret,
			},
		}
		return nil
	}
}

func WithTLSIssuer(issuerName string) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		if len(issuerName) == 0 {
			return errors.Errorf("annotation %s for %s must be present and not null, check your Fleet or GameServer manifest.", gameserver.OctopsAnnotationIssuerName, gs.Name)
		}

		ingress.Annotations = map[string]string{
			gameserver.CertManagerAnnotationIssuer: issuerName,
		}

		return nil
	}
}

func WithRules(mode IngressRoutingMode) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		var fqdn, path string

		switch mode {
		case IngressRoutingModePath:
			fqdn, path = gs.Annotations[gameserver.OctopsAnnotationIngressFQDN], gs.Name
		case IngressRoutingModeDomain:
			fallthrough
		default:
			fqdn, path = fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain]), "/"
		}

		rules := newIngressPathRules(fqdn, path, gs.Name, gameserver.GetGameServerContainerPort(gs))
		ingress.Spec.Rules = rules

		return nil
	}
}

func NewIngress(gs *agonesv1.GameServer, options ...IngressOption) (*networkingv1.Ingress, error) {
	if gs == nil {
		return nil, errors.New("gameserver can't be nil")
	}

	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
	ig := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: gs.Name,
			Labels: map[string]string{
				gameserver.AgonesGameServerNameLabel: gs.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
	}

	for _, opt := range options {
		if err := opt(gs, ig); err != nil {
			return nil, err
		}
	}

	return ig, nil
}

func newIngressPathRules(host, path, name string, port int32) []networkingv1.IngressRule {
	return []networkingv1.IngressRule{
		{
			Host: host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path:     path,
							PathType: &defaultPathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: name,
									Port: networkingv1.ServiceBackendPort{
										Number: port,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newIngressTLS(fqdn, secretName string) []networkingv1.IngressTLS {
	return []networkingv1.IngressTLS{
		{
			Hosts: []string{
				strings.TrimSpace(fqdn),
			},
			SecretName: fmt.Sprintf("%s-tls", strings.TrimSpace(secretName)),
		},
	}
}
