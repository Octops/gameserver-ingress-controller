package reconcilers

import (
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

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
