package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_MustReconcile(t *testing.T) {
	testCase := []struct {
		name        string
		annotations map[string]string
		wantErr     bool
		expected    bool
	}{
		{
			name: "Should not reconcile with ready Ingress",
			annotations: map[string]string{
				gameserver.OctopsAnnotationGameServerIngressReady: "true",
			},
			wantErr:  false,
			expected: false,
		},
		{
			name: "Should reconcile with not ready Ingress",
			annotations: map[string]string{
				gameserver.OctopsAnnotationGameServerIngressReady: "false",
			},
			wantErr:  false,
			expected: true,
		},
		{
			name: "Should not reconcile with multiple annotation",
			annotations: map[string]string{
				gameserver.OctopsAnnotationGameServerIngressReady: "true",
				"another-annotation":                              "another-value",
			},
			wantErr:  false,
			expected: false,
		},
		{
			name:        "Should reconcile without annotation",
			annotations: map[string]string{},
			wantErr:     false,
			expected:    true,
		},
		{
			name: "Should reconcile with wrong annotation",
			annotations: map[string]string{
				gameserver.OctopsAnnotationGameServerIngressReady: "123",
			},
			wantErr:  true,
			expected: true,
		},
		{
			name: "Should reconcile with empty annotation",
			annotations: map[string]string{
				gameserver.OctopsAnnotationGameServerIngressReady: "",
			},
			wantErr:  false,
			expected: true,
		},
		{
			name: "Should reconcile with multiple annotation",
			annotations: map[string]string{
				gameserver.OctopsAnnotationGameServerIngressReady: "",
				"another-annotation":                              "another-value",
			},
			wantErr:  false,
			expected: true,
		},
	}
	reconciler := &GameServerReconciler{}
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			gs := &agonesv1.GameServer{}
			gs.Annotations = tc.annotations

			got, err := reconciler.MustReconcile(gs)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expected, got)
		})
	}
}
