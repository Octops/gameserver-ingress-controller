package reconcilers

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_WithCustomAnnotations(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		expected    map[string]string
		notExpected map[string]string
		wantErr     bool
		err         error
	}{
		{
			name: "with single custom annotation",
			annotations: map[string]string{
				"octops.io/ingress-annotation-my-annotation": "my_custom_annotation_value",
			},
			expected: map[string]string{
				"my-annotation": "my_custom_annotation_value",
			},
			wantErr: false,
		},
		{
			name: "with two custom annotations",
			annotations: map[string]string{
				"octops.io/ingress-annotation-my-annotation-one": "my_custom_annotation_value_one",
				"octops.io/ingress-annotation-my-annotation-two": "my_custom_annotation_value_two",
			},
			expected: map[string]string{
				"my-annotation-one": "my_custom_annotation_value_one",
				"my-annotation-two": "my_custom_annotation_value_two",
			},
			wantErr: false,
		},
		{
			name: "return only one custom annotation",
			annotations: map[string]string{
				"octops.io/ingress-annotation-my-annotation-one": "my_custom_annotation_value_one",
				"octops.io/another-annotation":                   "another_annotation_value",
			},
			expected: map[string]string{
				"my-annotation-one": "my_custom_annotation_value_one",
			},
			notExpected: map[string]string{
				"octops.io/another-annotation": "another_annotation_value",
			},
			wantErr: false,
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
