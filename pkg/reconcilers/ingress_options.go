package reconcilers

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
)

type IngressOption func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error

func WithCustomAnnotationsTemplate() IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		data := struct {
			Name string
			Port int32
		}{
			Name: gs.Name,
			Port: gameserver.GetGameServerPort(gs).Port,
		}

		annotations := ingress.Annotations
		for k, v := range gs.Annotations {
			if strings.HasPrefix(k, gameserver.OctopsAnnotationCustomPrefix) {
				custom := strings.TrimPrefix(k, gameserver.OctopsAnnotationCustomPrefix)
				if len(custom) == 0 {
					return errors.New("custom annotation does not contain a suffix")
				}

				if !strings.Contains(v, "{{") || !strings.Contains(v, "}}") {
					continue
				}

				t, err := template.New("gs").Parse(v)
				if err != nil {
					return errors.Errorf("%s:%s does not contain a valid template", custom, v)
				}

				b := new(strings.Builder)
				err = t.Execute(b, data)
				if parsed := b.String(); len(parsed) > 0 {
					annotations[custom] = parsed
				}
			}
		}

		ingress.SetAnnotations(annotations)

		return nil
	}
}

func WithCustomAnnotations() IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		annotations := ingress.Annotations
		for k, v := range gs.Annotations {
			if strings.HasPrefix(k, gameserver.OctopsAnnotationCustomPrefix) {
				custom := strings.TrimPrefix(k, gameserver.OctopsAnnotationCustomPrefix)
				if len(custom) == 0 {
					return errors.New("custom annotation does not contain a suffix")
				}
				annotations[custom] = v
			}
		}

		ingress.SetAnnotations(annotations)
		return nil
	}
}

func WithTLS(mode gameserver.IngressRoutingMode) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		errMsgInvalidAnnotation := func(mode, annotation string) error {
			return errors.Errorf(gameserver.ErrIngressRoutingModeEmpty, mode, annotation)
		}

		secret, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationsTLSSecretName)
		if ok && len(secret) == 0 {
			return errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, gs.Namespace, gs.Name, gameserver.OctopsAnnotationsTLSSecretName)
		}

		tlsForDomain := func(gs *agonesv1.GameServer) ([]networkingv1.IngressTLS, error) {
			domain, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressDomain)
			if !ok {
				return []networkingv1.IngressTLS{}, errMsgInvalidAnnotation(mode.String(), gameserver.OctopsAnnotationIngressDomain)
			}

			domains := strings.Split(domain, ",")
			tls := make([]networkingv1.IngressTLS, len(domains))
			for i, d := range domains {
				tlsSecret := secret
				if len(secret) == 0 {
					tlsSecret = strings.ReplaceAll(fmt.Sprintf("%s-%s-tls", d, gs.Name), ".", "-")
				}

				tls[i] = networkingv1.IngressTLS{
					Hosts: []string{
						fmt.Sprintf("%s.%s", gs.Name, d),
					},
					SecretName: tlsSecret,
				}
			}

			return tls, nil
		}

		tlsForPath := func(gs *agonesv1.GameServer) ([]networkingv1.IngressTLS, error) {
			fqdn, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressFQDN)
			if !ok {
				return []networkingv1.IngressTLS{}, errMsgInvalidAnnotation(mode.String(), gameserver.OctopsAnnotationIngressFQDN)
			}

			fqdns := strings.Split(fqdn, ",")
			tls := make([]networkingv1.IngressTLS, len(fqdns))
			for i, f := range fqdns {
				tlsSecret := secret
				if len(secret) == 0 {
					tlsSecret = strings.ReplaceAll(fmt.Sprintf("%s-%s-tls", f, gs.Name), ".", "-")
				}

				tls[i] = networkingv1.IngressTLS{
					Hosts: []string{
						fmt.Sprintf("%s.%s", gs.Name, f),
					},
					SecretName: tlsSecret,
				}
			}

			return tls, nil
		}

		var err error
		var tls []networkingv1.IngressTLS

		switch mode {
		case gameserver.IngressRoutingModePath:
			tls, err = tlsForPath(gs)
		case gameserver.IngressRoutingModeDomain:
			fallthrough
		default:
			tls, err = tlsForDomain(gs)
		}

		if err != nil {
			return err
		}

		ingress.Spec.TLS = tls

		return nil
	}
}

func WithIngressRule(mode gameserver.IngressRoutingMode) IngressOption {
	return func(gs *agonesv1.GameServer, ingress *networkingv1.Ingress) error {
		errMsgInvalidAnnotation := func(mode, annotation, gsName string) error {
			return errors.Errorf("ingress routing mode %s requires the annotation %s to be present on %s, check your Fleet or GameServer manifest.", mode, annotation, gsName)
		}

		var rules []networkingv1.IngressRule

		switch mode {
		case gameserver.IngressRoutingModePath:
			fqdns, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressFQDN)
			if !ok {
				return errMsgInvalidAnnotation(mode.String(), gameserver.OctopsAnnotationIngressFQDN, gs.Name)
			}

			for _, f := range strings.Split(fqdns, ",") {
				rule := newIngressRule(f, "/"+gs.Name, gs.Name, gameserver.GetGameServerPort(gs).Port)
				rules = append(rules, rule)
			}
		case gameserver.IngressRoutingModeDomain:
			domains, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressDomain)
			if !ok {
				return errMsgInvalidAnnotation(mode.String(), gameserver.OctopsAnnotationIngressDomain, gs.Name)
			}

			for _, d := range strings.Split(domains, ",") {
				host := fmt.Sprintf("%s.%s", gs.Name, d)
				rule := newIngressRule(host, "/", gs.Name, gameserver.GetGameServerPort(gs).Port)
				rules = append(rules, rule)
			}
		default:
			return errors.Errorf("routing mode not recognised: %s", mode)
		}

		ingress.Spec.Rules = rules
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
			return errors.Errorf("annotation %s for %s must be present, check your Fleet or GameServer manifest.", gameserver.OctopsAnnotationIssuerName, gs.Name)
		}

		ingress.Annotations[gameserver.CertManagerAnnotationIssuer] = issuerName
		return nil
	}
}

func newIngressRule(host, path, serviceName string, port int32) networkingv1.IngressRule {
	return networkingv1.IngressRule{
		Host: strings.TrimSpace(host),
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     path,
						PathType: &defaultPathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: serviceName,
								Port: networkingv1.ServiceBackendPort{
									Number: port,
								},
							},
						},
					},
				},
			},
		},
	}
}
