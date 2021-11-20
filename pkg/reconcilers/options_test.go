package reconcilers

import (
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
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
				newCustomAnnotation("nginx.ingress.kubernetes.io/proxy-read-timeout"): "10",
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "10",
			},
		},
		{
			name: "with multiline annotation",
			annotations: map[string]string{
				newCustomAnnotation("nginx.ingress.kubernetes.io/server-snippet"): `|
        set $agentflag 0;

        if ($http_user_agent ~* "(Mobile)" ){
          set $agentflag 1;
        }

        if ( $agentflag = 1 ) {
          return 301 https://m.example.com;
        }`,
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/server-snippet": `|
        set $agentflag 0;

        if ($http_user_agent ~* "(Mobile)" ){
          set $agentflag 1;
        }

        if ( $agentflag = 1 ) {
          return 301 https://m.example.com;
        }`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := newGameServer(tc.annotations)

			ingress, err := newIngress(gs, WithCustomAnnotations())
			require.NoError(t, err)

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
