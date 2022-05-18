package reconcilers

import (
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
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
			name:           "with mixed annotations with template",
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
		expected    string
		wantErr     bool
	}{
		{
			name:   "with custom secret name for domain mode",
			gsName: "simple-gameserver-with-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain:  "example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "my_custom_secret_name",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    "my_custom_secret_name",
		},
		{
			name:   "with custom secret name for path mode",
			gsName: "simple-gameserver-with-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN:    "www.example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "my_custom_secret_name",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected:    "my_custom_secret_name",
		},
		{
			name:   "no custom secret name for domain mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain: "example.com",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    "simple-gameserver-no-custom-tls",
		},
		{
			name:   "no custom secret name for path mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressFQDN: "www.example.com",
			},
			routingMode: gameserver.IngressRoutingModePath,
			expected:    "simple-gameserver-no-custom-tls",
		},
		{
			name:   "empty secret annotation for domain mode",
			gsName: "simple-gameserver-no-custom",
			annotations: map[string]string{
				gameserver.OctopsAnnotationIngressDomain:  "example.com",
				gameserver.OctopsAnnotationsTLSSecretName: "",
			},
			routingMode: gameserver.IngressRoutingModeDomain,
			expected:    "",
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
			expected:    "",
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
				require.Equal(t, errors.Errorf(gameserver.ErrGameServerAnnotationEmpty, gs.Namespace, gs.Name, gameserver.OctopsAnnotationsTLSSecretName).Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, ingress)
				require.Equal(t, tc.expected, ingress.Spec.TLS[0].SecretName)
			}

		})
	}
}
