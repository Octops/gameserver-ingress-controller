package stores

import (
	"context"

	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayinformersv1 "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions/apis/v1"
)

type gatewayStore struct {
	client   gatewayclient.Interface
	informer gatewayinformersv1.HTTPRouteInformer
}

func newGatewayStore(client gatewayclient.Interface, informer gatewayinformersv1.HTTPRouteInformer) *gatewayStore {
	return &gatewayStore{client: client, informer: informer}
}

func (s *gatewayStore) CreateHTTPRoute(ctx context.Context, route *gatewayv1.HTTPRoute, options metav1.CreateOptions) (*gatewayv1.HTTPRoute, error) {
	result, err := s.client.GatewayV1().HTTPRoutes(route.Namespace).Create(ctx, route, options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create HTTPRoute %s", k8sutil.Namespaced(route))
	}

	return result, nil
}

func (s *gatewayStore) GetHTTPRoute(name, namespace string) (*gatewayv1.HTTPRoute, error) {
	result, err := s.informer.Lister().HTTPRoutes(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, err
		}

		return nil, errors.Wrapf(err, "error retrieving HTTPRoute %s from namespace %s", name, namespace)
	}

	return result, nil
}
