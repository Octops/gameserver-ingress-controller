package stores

import (
	"context"

	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
)

type ingressStore struct {
	client   kubernetes.Interface
	informer networkinginformers.IngressInformer
}

func newIngressStore(client kubernetes.Interface, informer networkinginformers.IngressInformer) *ingressStore {
	return &ingressStore{client: client, informer: informer}
}

func (s *ingressStore) CreateIngress(ctx context.Context, ingress *networkingv1.Ingress, options metav1.CreateOptions) (*networkingv1.Ingress, error) {
	result, err := s.client.NetworkingV1().Ingresses(ingress.Namespace).Create(ctx, ingress, options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Ingress %s", k8sutil.Namespaced(ingress))
	}

	return result, nil
}

func (s *ingressStore) GetIngress(name, namespace string) (*networkingv1.Ingress, error) {
	result, err := s.informer.Lister().Ingresses(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, err
		}

		return nil, errors.Wrapf(err, "error retrieving Ingress %s from namespace %s", name, namespace)
	}

	return result, nil
}
