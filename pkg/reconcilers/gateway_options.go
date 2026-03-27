package reconcilers

import (
	"fmt"
	"strings"
	"text/template"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type HTTPRouteOption func(gs *agonesv1.GameServer, route *gatewayv1.HTTPRoute) error

func WithHTTPRouteParentRef() HTTPRouteOption {
	return func(gs *agonesv1.GameServer, route *gatewayv1.HTTPRoute) error {
		name, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationGatewayName)
		if !ok || len(name) == 0 {
			return errors.Errorf(gameserver.ErrGameServerAnnotationMissing, gs.Namespace, gs.Name, gameserver.OctopsAnnotationGatewayName)
		}

		ref := gatewayv1.ParentReference{
			Name: gatewayv1.ObjectName(name),
		}

		if ns, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationGatewayNamespace); ok && len(ns) > 0 {
			gwNs := gatewayv1.Namespace(ns)
			ref.Namespace = &gwNs
		}

		if section, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationGatewaySectionName); ok && len(section) > 0 {
			sectionName := gatewayv1.SectionName(section)
			ref.SectionName = &sectionName
		}

		route.Spec.ParentRefs = []gatewayv1.ParentReference{ref}
		return nil
	}
}

func WithHTTPRouteRules(mode gameserver.IngressRoutingMode) HTTPRouteOption {
	return func(gs *agonesv1.GameServer, route *gatewayv1.HTTPRoute) error {
		port := gatewayv1.PortNumber(gameserver.GetGameServerPort(gs).Port)
		backendRef := gatewayv1.HTTPBackendRef{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(gs.Name),
					Port: &port,
				},
			},
		}

		pathPrefix := gatewayv1.PathMatchPathPrefix
		var hostnames []gatewayv1.Hostname
		var pathValue string

		switch mode {
		case gameserver.IngressRoutingModePath:
			fqdns, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressFQDN)
			if !ok {
				return errors.Errorf(gameserver.ErrGameServerAnnotationMissing, gs.Namespace, gs.Name, gameserver.OctopsAnnotationIngressFQDN)
			}
			if len(fqdns) == 0 {
				return errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, gs.Namespace, gs.Name, gameserver.OctopsAnnotationIngressFQDN)
			}
			for _, f := range strings.Split(fqdns, ",") {
				hostnames = append(hostnames, gatewayv1.Hostname(strings.TrimSpace(f)))
			}
			pathValue = "/" + gs.Name

		case gameserver.IngressRoutingModeDomain:
			domains, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressDomain)
			if !ok {
				return errors.Errorf(gameserver.ErrGameServerAnnotationMissing, gs.Namespace, gs.Name, gameserver.OctopsAnnotationIngressDomain)
			}
			if len(domains) == 0 {
				return errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, gs.Namespace, gs.Name, gameserver.OctopsAnnotationIngressDomain)
			}
			for _, d := range strings.Split(domains, ",") {
				hostnames = append(hostnames, gatewayv1.Hostname(fmt.Sprintf("%s.%s", gs.Name, strings.TrimSpace(d))))
			}
			pathValue = "/"

		default:
			return errors.Errorf("routing mode '%s' from gameserver %s/%s is not recognised", mode, gs.Namespace, gs.Name)
		}

		route.Spec.Hostnames = hostnames
		route.Spec.Rules = []gatewayv1.HTTPRouteRule{
			{
				Matches: []gatewayv1.HTTPRouteMatch{
					{
						Path: &gatewayv1.HTTPPathMatch{
							Type:  &pathPrefix,
							Value: &pathValue,
						},
					},
				},
				BackendRefs: []gatewayv1.HTTPBackendRef{backendRef},
			},
		}

		return nil
	}
}

func WithCustomHTTPRouteAnnotations() HTTPRouteOption {
	return func(gs *agonesv1.GameServer, route *gatewayv1.HTTPRoute) error {
		annotations := route.Annotations
		for k, v := range gs.Annotations {
			if strings.HasPrefix(k, gameserver.OctopsAnnotationCustomPrefix) {
				custom := strings.TrimPrefix(k, gameserver.OctopsAnnotationCustomPrefix)
				if len(custom) == 0 {
					return errors.New("custom annotation does not contain a suffix")
				}
				annotations[custom] = v
			}
		}
		route.SetAnnotations(annotations)
		return nil
	}
}

func WithCustomHTTPRouteAnnotationsTemplate() HTTPRouteOption {
	return func(gs *agonesv1.GameServer, route *gatewayv1.HTTPRoute) error {
		data := struct {
			Name string
			Port int32
		}{
			Name: gs.Name,
			Port: gameserver.GetGameServerPort(gs).Port,
		}

		annotations := route.Annotations
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
				_ = t.Execute(b, data)
				if parsed := b.String(); len(parsed) > 0 {
					annotations[custom] = parsed
				}
			}
		}

		route.SetAnnotations(annotations)
		return nil
	}
}
