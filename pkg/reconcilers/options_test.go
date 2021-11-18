package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewIngressForDomainRoutingMode(t *testing.T) {
	t.Run("ingress for domain routing mode", func(t *testing.T) {
		domain := "foo.bar"
		gs := newGameServer(map[string]string{
			gameserver.OctopsAnnotationIngressDomain: domain,
		})

		fqdn := fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain])
		ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
		tls := newIngressTLS(fqdn, gs.Name)
		rules := newIngressRule(fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain]), "/", gs.Name, gameserver.GetGameServerPort(gs).Port)
		issuerName := "selfSigned"

		opts := []IngressOption{
			WithIngressRule(IngressRoutingModeDomain),
			WithTLS(IngressRoutingModeDomain),
			WithTLSIssuer(issuerName),
		}
		ig, err := NewIngress(gs, opts...)

		require.NoError(t, err)
		require.Equal(t, gs.Name, ig.Name)
		require.Contains(t, ig.Labels, gameserver.AgonesGameServerNameLabel)
		require.Equal(t, ig.Labels[gameserver.AgonesGameServerNameLabel], gs.Name)
		require.Equal(t, []metav1.OwnerReference{*ref}, ig.OwnerReferences)
		require.Equal(t, tls, ig.Spec.TLS)
		require.Equal(t, rules, ig.Spec.Rules)
		require.Contains(t, ig.Annotations, gameserver.CertManagerAnnotationIssuer)
		require.Equal(t, issuerName, ig.Annotations[gameserver.CertManagerAnnotationIssuer])
	})
}

func newGameServer(annotations map[string]string) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "simple-gameserver",
			Namespace:   "default",
			Annotations: annotations,
		},
	}
}
