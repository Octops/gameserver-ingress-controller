package reconcilers

import (
	"fmt"
	"strings"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func Test_WithCustomAnnotationsTemplate(t *testing.T) {
	testCase := []struct {
		name           string
		gameserverName string
		annotations    map[string]string
		wantErr        bool
		expected       map[string]string
	}{
		{
			name:           "with not custom annotations",
			gameserverName: "game-1",
			annotations: map[string]string{
				"annotation/not_custom": "somevalue",
			},
			wantErr:  false,
			expected: map[string]string{},
		},
		{
			name:           "with custom annotation without template",
			gameserverName: "game-2",
			annotations: map[string]string{
				"octops-annotation/custom": "somevalue",
			},
			wantErr:  false,
			expected: map[string]string{},
		},
		{
			name:           "with custom annotation with template only",
			gameserverName: "game-3",
			annotations: map[string]string{
				"octops-annotation/custom": "{{ .Name }}",
			},
			wantErr: false,
			expected: map[string]string{
				"annotation/custom": "game-3",
			},
		},
		{
			name:           "with custom annotation with complex template",
			gameserverName: "game-4",
			annotations: map[string]string{
				"octops-annotation/custom": "/{{ .Name }}",
			},
			wantErr: false,
			expected: map[string]string{
				"annotation/custom": "/game-4",
			},
		},
		{
			name:           "with custom annotation with invalid template",
			gameserverName: "game-5",
			annotations: map[string]string{
				"octops-annotation/custom": "}}{{",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name:           "with custom annotation with invalid template",
			gameserverName: "game-6",
			annotations: map[string]string{
				"octops-annotation/custom": "{{}}",
			},
			wantErr: true,
		},
		{
			name:           "with custom annotation with invalid mixed template",
			gameserverName: "game-7",
			annotations: map[string]string{
				"octops-annotation/custom": "{{ /.Name}}",
			},
			wantErr: true,
		},
		{
			name:           "with custom annotation with invalid field",
			gameserverName: "game-8",
			annotations: map[string]string{
				"octops-annotation/custom": "{{ .SomeField }}",
			},
			expected: map[string]string{},
			wantErr:  false,
		},
		{
			name:           "with not custom annotation with template",
			gameserverName: "game-9",
			annotations: map[string]string{
				"annotation/not-custom": "{{ .SomeField }}",
			},
			wantErr:  false,
			expected: map[string]string{},
		},
		{
			name:           "with custom envoy annotation with template",
			gameserverName: "game-10",
			annotations: map[string]string{
				"octops-projectcontour.io/websocket-routes": "/{{ .Name }}",
			},
			wantErr: false,
			expected: map[string]string{
				"projectcontour.io/websocket-routes": "/game-10",
			},
		},
		{
			name:           "with multiples annotations",
			gameserverName: "game-10",
			annotations: map[string]string{
				"annotation/not-custom":                     "somevalue",
				"octops-projectcontour.io/websocket-routes": "/{{ .Name }}",
			},
			wantErr: false,
			expected: map[string]string{
				"projectcontour.io/websocket-routes": "/game-10",
			},
		},
		{
			name:           "with multiples annotations inverted",
			gameserverName: "game-11",
			annotations: map[string]string{
				"octops-projectcontour.io/websocket-routes": "/{{ .Name }}",
				"annotation/not-custom":                     "somevalue",
			},
			wantErr: false,
			expected: map[string]string{
				"projectcontour.io/websocket-routes": "/game-11",
			},
		},
		{
			name:           "with multiples annotations with template",
			gameserverName: "game-12",
			annotations: map[string]string{
				"octops-projectcontour.io/websocket-routes": "/{{ .Name }}",
				"octops-annotation/custom":                  "custom-{{ .Name }}",
			},
			wantErr: false,
			expected: map[string]string{
				"projectcontour.io/websocket-routes": "/game-12",
				"annotation/custom":                  "custom-game-12",
			},
		},
		{
			name:           "with mixed annotations with Name template",
			gameserverName: "game-13",
			annotations: map[string]string{
				"annotation/not-custom":                     "some-value",
				"annotation/not-custom-template":            "some-{{ .Name }}",
				"octops-projectcontour.io/websocket-routes": "/{{ .Name }}",
				"octops-annotation/custom":                  "custom-{{ .Name }}",
			},
			wantErr: false,
			expected: map[string]string{
				"projectcontour.io/websocket-routes": "/game-13",
				"annotation/custom":                  "custom-game-13",
			},
		},
		{
			name:           "with mixed annotations with Port template",
			gameserverName: "game-13",
			annotations: map[string]string{
				"annotation/not-custom":                          "some-value",
				"annotation/not-custom-template":                 "some-{{ .Port }}",
				"octops-projectcontour.io/upstream-protocol.tls": "{{ .Port }}",
				"octops-annotation/custom":                       "custom-{{ .Port }}",
			},
			wantErr: false,
			expected: map[string]string{
				"projectcontour.io/upstream-protocol.tls": "7771",
				"annotation/custom":                       "custom-7771",
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			gs := newGameServer(tc.gameserverName, "default", tc.annotations)
			require.Equal(t, tc.gameserverName, gs.Name)

			ingress, err := newIngress(gs, WithCustomAnnotationsTemplate())
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, ingress)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, ingress.Annotations)
			}
		})
	}
}

func Test_WithCustomAnnotations(t *testing.T) {
	newCustomAnnotation := func(custom string) string {
		return fmt.Sprintf("%s%s", gameserver.OctopsAnnotationCustomPrefix, custom)
	}

	testCases := []struct {
		name        string
		annotations map[string]string
		expected    map[string]string
		notExpected map[string]string
		wantErr     bool
	}{
		{
			name: "with single custom annotation",
			annotations: map[string]string{
				newCustomAnnotation("my-annotation"): "my_custom_annotation_value",
			},
			expected: map[string]string{
				"my-annotation": "my_custom_annotation_value",
			},
		},
		{
			name: "with two custom annotations",
			annotations: map[string]string{
				newCustomAnnotation("my-annotation-one"): "my_custom_annotation_value_one",
				newCustomAnnotation("my-annotation-two"): "my_custom_annotation_value_two",
			},
			expected: map[string]string{
				"my-annotation-one": "my_custom_annotation_value_one",
				"my-annotation-two": "my_custom_annotation_value_two",
			},
		},
		{
			name: "return only one custom annotation",
			annotations: map[string]string{
				newCustomAnnotation("my-annotation-one"): "my_custom_annotation_value_one",
				"octops.io/another-annotation":           "another_annotation_value",
			},
			expected: map[string]string{
				"my-annotation-one": "my_custom_annotation_value_one",
			},
			notExpected: map[string]string{
				"octops.io/another-annotation": "another_annotation_value",
			},
		},
		{
			name: "with complex annotations",
			annotations: map[string]string{
				newCustomAnnotation("haproxy.org/rate-limit-status-code"): "429",
			},
			expected: map[string]string{
				"haproxy.org/rate-limit-status-code": "429",
			},
		},
		{
			name: "with multiline annotation",
			annotations: map[string]string{
				newCustomAnnotation("haproxy.org/backend-config-snippet"): `|
        http-send-name-header x-dst-server
        stick-table type string len 32 size 100k expire 30m
        stick on req.cook(sessionid)`,
			},
			expected: map[string]string{
				"haproxy.org/backend-config-snippet": `|
        http-send-name-header x-dst-server
        stick-table type string len 32 size 100k expire 30m
        stick on req.cook(sessionid)`,
			},
		},
		{
			name: "error with annotations prefix only",
			annotations: map[string]string{
				newCustomAnnotation(""): "anyValue",
			},
			expected: map[string]string{},
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := newGameServer("simple-gameserver", "default", tc.annotations)

			ingress, err := newIngress(gs, WithCustomAnnotations())
			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, "custom annotation does not contain a suffix", err.Error())
			} else {
				require.NoError(t, err)
			}

			for k, v := range tc.expected {
				value, ok := ingress.Annotations[k]
				require.True(t, ok, "annotations %s is not present", k)
				require.Equal(t, v, value)
			}

			for k, _ := range tc.notExpected {
				require.NotContains(t, ingress.Annotations, k, "annotations %s should not present", k)
			}
		})
	}
}

func Test_WithTLS(t *testing.T) {
	testCases := []struct {
		name        string
		gsName      string
		annotations map[string]string
		routingMode gameserver.IngressRoutingMode
		expected    []networkingv1.IngressTLS
		wantErr     bool
		err         error
	}{
		{
			name:   "with custom secret name for domain mode",
			gsName: "simple-gameserver-with-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain:  "example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "my_custom_secret_name",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"simple-gameserver-with-custom.example.com"},
					SecretName: "my_custom_secret_name",
				},
			},
		},
		{
			name:   "with custom secret name for path mode",
			gsName: "simple-gameserver-with-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN:    "www.example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "my_custom_secret_name",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"www.example.com"},
					SecretName: "my_custom_secret_name",
				},
			},
		},
		{
			name:   "no custom secret name for domain mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain: "example.com",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"simple-gameserver-no-custom.example.com"},
					SecretName: "example-com-simple-gameserver-no-custom-tls",
				},
			},
		},
		{
			name:   "no custom secret name for domain mode with multiple domains",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain: "example.com,example.gg",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"simple-gameserver-no-custom.example.com"},
					SecretName: "example-com-simple-gameserver-no-custom-tls",
				},
				{
					Hosts:      []string{"simple-gameserver-no-custom.example.gg"},
					SecretName: "example-gg-simple-gameserver-no-custom-tls",
				},
			},
		},
		{
			name:   "no custom secret name for path mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN: "www.example.com",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"www.example.com"},
					SecretName: "www-example-com-simple-gameserver-no-custom-tls",
				},
			},
		},
		{
			name:   "no custom secret name for path mode with multiple domains",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN: "www.example.com,www.example.gg",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"www.example.com"},
					SecretName: "www-example-com-simple-gameserver-no-custom-tls",
				},
				{
					Hosts:      []string{"www.example.gg"},
					SecretName: "www-example-gg-simple-gameserver-no-custom-tls",
				},
			},
		},
		{
			name:   "empty secret annotation for domain mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain:  "example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    []networkingv1.IngressTLS{},
			err:         errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, "default", "simple-gameserver-no-custom", gameserver.OctopsAnnotationsTLSSecretName),
			wantErr:     true,
		},
		{
			name:   "empty secret annotation for path mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN:    "www.example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected:    []networkingv1.IngressTLS{},
			err:         errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, "default", "simple-gameserver-no-custom", gameserver.OctopsAnnotationsTLSSecretName),
			wantErr:     true,
		},
		{
			name:        "error routing mode empty for path mode",
			gsName:      "simple-gameserver-no-custom",
			annotations: map[string]string{},
			routingMode: gameserver.IngressRoutingModePath,
			expected:    []networkingv1.IngressTLS{},
			err:         errors.Errorf(gameserver.ErrIngressRoutingModeEmpty, gameserver.IngressRoutingModePath, gameserver.OctopsAnnotationIngressFQDN, "default", "simple-gameserver-no-custom"),
			wantErr:     true,
		},
		{
			name:        "error routing mode empty for domain mode",
			gsName:      "simple-gameserver-no-custom",
			annotations: map[string]string{},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    []networkingv1.IngressTLS{},
			err:         errors.Errorf(gameserver.ErrIngressRoutingModeEmpty, gameserver.IngressRoutingModeDomain, gameserver.OctopsAnnotationIngressDomain, "default", "simple-gameserver-no-custom"),
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := newGameServer(tc.gsName, "default", tc.annotations)

			ingress, err := newIngress(gs, WithTLS(tc.routingMode))
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, ingress)
				require.Equal(t, tc.err.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, ingress)
				require.Equal(t, tc.expected, ingress.Spec.TLS)
			}
		})
	}
}

func Test_WithIngressRule(t *testing.T) {
	testCase := map[string]struct {
		gsName            string
		annotations       map[string]string
		routingMode       gameserver.IngressRoutingMode
		missingAnnotation string
		expected          []networkingv1.IngressRule
		wantErr           bool
	}{
		"routing mode path with single domain": {
			gsName: "test-game-server",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN: "www.example.com",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected:    newIngressRules("www.example.com", "/test-game-server", "test-game-server", 7771),
			wantErr:     false,
		},
		"routing mode path with multiple domain": {
			gsName: "test-game-server",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN: "www.example.com,www.example.gg",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected:    newIngressRules("www.example.com,www.example.gg", "/test-game-server", "test-game-server", 7771),
			wantErr:     false,
		},
		"routing mode path with error": {
			gsName: "test-game-server",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN: "",
			},
			missingAnnotation: gameserver.OctopsAnnotationIngressFQDN,
			routingMode:       gameserver.IngressRoutingModePath,
			wantErr:           true,
		},
		"routing mode domain with single domain": {
			gsName: "test-game-server",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain: "example.com",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    newIngressRules("test-game-server.example.com", "/", "test-game-server", 7771),
			wantErr:     false,
		},
		"routing mode domain with multiple domain": {
			gsName: "test-game-server",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain: "example.com,example.gg",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    newIngressRules("test-game-server.example.com,test-game-server.example.gg", "/", "test-game-server", 7771),
			wantErr:     false,
		},
		"routing mode domain with error": {
			gsName: "test-game-server",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain: "",
			},
			missingAnnotation: gameserver.OctopsAnnotationIngressDomain,
			routingMode:       gameserver.IngressRoutingModeDomain,
			wantErr:           true,
		},
	}

	for name, tc := range testCase {
		t.Run(name, func(t *testing.T) {
			gs := newGameServer(tc.gsName, "default", tc.annotations)

			ingress, err := newIngress(gs, WithIngressRule(tc.routingMode))
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, ingress)
				require.Equal(t, errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, gs.Namespace, gs.Name, tc.missingAnnotation).Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, ingress)
				require.Equal(t, tc.expected, ingress.Spec.Rules)
			}
		})
	}
}

func newIngressRules(hosts, path, svcName string, port int32) []networkingv1.IngressRule {
	var rules []networkingv1.IngressRule

	for _, host := range strings.Split(hosts, ",") {
		rule := networkingv1.IngressRule{
			Host: strings.TrimSpace(host),
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path:     path,
							PathType: &defaultPathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: svcName,
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
		rules = append(rules, rule)
	}

	return rules
}
