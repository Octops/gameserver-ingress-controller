package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	"strconv"
	"strings"
)

type IngressOption func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error

func WithTLS(mode gameserver.IngressRoutingMode) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		errMsgInvalidAnnotation := func(mode, annotation string) error {
			return errors.Errorf("ingress routing mode %s requires the annotation %s to be set", mode, annotation)
		}

		tlsForDomain := func(gs *agonesv1.GameServer) (fqdn, secretName string, err error) {
			if value, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressDomain); !ok {
				return "", "", errMsgInvalidAnnotation(value, gameserver.OctopsAnnotationIngressDomain)
			}

			return fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain]), fmt.Sprintf("%s-tls", gs.Name), nil
		}

		tlsForPath := func(gs *agonesv1.GameServer) (fqdn, secretName string, err error) {
			if value, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressFQDN); !ok {
				return "", "", errMsgInvalidAnnotation(value, gameserver.OctopsAnnotationIngressFQDN)
			}

			return gs.Annotations[gameserver.OctopsAnnotationIngressFQDN], fmt.Sprintf("%s-tls", gs.Name), nil
		}

		var host, secret string
		var err error

		switch mode {
		case gameserver.IngressRoutingModePath:
			host, secret, err = tlsForPath(gs)
		case gameserver.IngressRoutingModeDomain:
			fallthrough
		default:
			host, secret, err = tlsForDomain(gs)
		}

		if err != nil {
			return err
		}

		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts: []string{
					host,
				},
				SecretName: secret,
			},
		}

		return nil
	}
}

func WithIngressRule(mode gameserver.IngressRoutingMode) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		var host, path string

		switch mode {
		case gameserver.IngressRoutingModePath:
			host, path = gs.Annotations[gameserver.OctopsAnnotationIngressFQDN], "/"+gs.Name
		case gameserver.IngressRoutingModeDomain:
			fallthrough
		default:
			host, path = fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain]), "/"
		}

		ingress.Spec.Rules = newIngressRule(host, path, gs.Name, gameserver.GetGameServerPort(gs).Port)

		return nil
	}
}

func WithTLSCertIssuer(issuerName string) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		terminate, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationTerminateTLS)
		if !ok || len(terminate) == 0 {
			return nil
		}

		if terminateTLS, err := strconv.ParseBool(terminate); err != nil {
			return errors.Errorf("annotation %s for %s must be \"true\" or \"false\"", gameserver.OctopsAnnotationTerminateTLS, gs.Name)
		} else if terminateTLS == false {
			return nil
		}

		if len(issuerName) == 0 {
			return errors.Errorf("annotation %s for %s must be present and not null, check your Fleet or GameServer manifest.", gameserver.OctopsAnnotationIssuerName, gs.Name)
		}

		ingress.Annotations = map[string]string{
			gameserver.CertManagerAnnotationIssuer: issuerName,
		}

		return nil
	}
}

func newIngressRule(host, path, name string, port int32) []networkingv1.IngressRule {
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

func newIngressTLS(host, secretName string) []networkingv1.IngressTLS {
	return []networkingv1.IngressTLS{
		{
			Hosts: []string{
				strings.TrimSpace(host),
			},
			SecretName: fmt.Sprintf("%s-tls", strings.TrimSpace(secretName)),
		},
	}
}
