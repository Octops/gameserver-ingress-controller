package reconcilers

import (
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_WithCustomServiceAnnotationsTemplate(t *testing.T) {
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
				"octops.service-annotation/custom": "somevalue",
			},
			wantErr:  false,
			expected: map[string]string{},
		},
		{
			name:           "with custom annotation with template only",
			gameserverName: "game-3",
			annotations: map[string]string{
				"octops.service-annotation/custom": "{{ .Name }}",
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
				"octops.service-annotation/custom": "/{{ .Name }}",
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
				"octops.service-annotation/custom": "}}{{",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name:           "with custom annotation with invalid template",
			gameserverName: "game-6",
			annotations: map[string]string{
				"octops.service-annotation/custom": "{{}}",
			},
			wantErr: true,
		},
		{
			name:           "with custom annotation with invalid mixed template",
			gameserverName: "game-7",
			annotations: map[string]string{
				"octops.service-annotation/custom": "{{ /.Name}}",
			},
			wantErr: true,
		},
		{
			name:           "with custom annotation with invalid field",
			gameserverName: "game-8",
			annotations: map[string]string{
				"octops.service-annotation/custom": "{{ .SomeField }}",
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
				"octops.service-projectcontour.io/websocket-routes": "/{{ .Name }}",
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
				"annotation/not-custom":                             "somevalue",
				"octops.service-projectcontour.io/websocket-routes": "/{{ .Name }}",
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
				"octops.service-projectcontour.io/websocket-routes": "/{{ .Name }}",
				"annotation/not-custom":                             "somevalue",
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
				"octops.service-projectcontour.io/websocket-routes": "/{{ .Name }}",
				"octops.service-annotation/custom":                  "custom-{{ .Name }}",
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
				"annotation/not-custom":                             "some-value",
				"annotation/not-custom-template":                    "some-{{ .Name }}",
				"octops.service-projectcontour.io/websocket-routes": "/{{ .Name }}",
				"octops.service-annotation/custom":                  "custom-{{ .Name }}",
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
				"annotation/not-custom":                                  "some-value",
				"annotation/not-custom-template":                         "some-{{ .Port }}",
				"octops.service-projectcontour.io/upstream-protocol.tls": "{{ .Port }}",
				"octops.service-annotation/custom":                       "custom-{{ .Port }}",
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

			service, err := newService(gs, WithCustomServiceAnnotationsTemplate())
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, service)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, service.Annotations)
			}
		})
	}
}

func Test_WithCustomServiceAnnotations(t *testing.T) {
	newCustomAnnotation := func(custom string) string {
		return fmt.Sprintf("%s%s", gameserver.OctopsAnnotationCustomServicePrefix, custom)
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

			service, err := newService(gs, WithCustomServiceAnnotations())
			if tc.wantErr {
				require.Error(t, err)
				require.Equal(t, "custom annotation "+gameserver.OctopsAnnotationCustomServicePrefix+" does not contain a suffix", err.Error())
			} else {
				require.NoError(t, err)
			}

			for k, v := range tc.expected {
				value, ok := service.Annotations[k]
				require.True(t, ok, "annotations %s is not present", k)
				require.Equal(t, v, value)
			}

			for k, _ := range tc.notExpected {
				require.NotContains(t, service.Annotations, k, "annotations %s should not present", k)
			}
		})
	}
}
