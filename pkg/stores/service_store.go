package stores

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
)

type serviceStore struct {
	client   kubernetes.Interface
	informer coreinformers.ServiceInformer
}

func newServiceStore(client kubernetes.Interface, informer coreinformers.ServiceInformer) *serviceStore {
	return &serviceStore{client: client, informer: informer}
}

func (s *serviceStore) CreateService(ctx context.Context, service *corev1.Service, options metav1.CreateOptions) (*corev1.Service, error) {
	result, err := s.client.CoreV1().Services(service.Namespace).Create(ctx, service, options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Service %s", k8sutil.Namespaced(service))
	}

	return result, nil
}

func (s *serviceStore) GetService(name, namespace string) (*corev1.Service, error) {
	result, err := s.informer.Lister().Services(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, err
		}

		return nil, errors.Wrapf(err, "error retrieving Service %s from namespace %s", name, namespace)
	}

	return result, nil
}
