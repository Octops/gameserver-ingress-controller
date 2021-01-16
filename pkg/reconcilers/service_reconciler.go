package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type ServiceReconciler struct {
	logger *logrus.Entry
	Client *kubernetes.Clientset
}

func NewServiceReconciler(client *kubernetes.Clientset) *ServiceReconciler {
	return &ServiceReconciler{
		logger: runtime.Logger().WithField("role", "service_reconciler"),
		Client: client,
	}
}

func (r *ServiceReconciler) Reconcile(gs *agonesv1.GameServer) (*corev1.Service, error) {
	service, err := r.Client.CoreV1().Services(gs.Namespace).Get(gs.Name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return &corev1.Service{}, errors.Wrapf(err, "error retrieving Service %s from namespace %s", gs.Name, gs.Namespace)
		}

		return r.reconcileNotFound(gs)
	}

	return service, nil
}

func (r *ServiceReconciler) reconcileNotFound(gs *agonesv1.GameServer) (*corev1.Service, error) {
	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gs.Name,
			Namespace: gs.GetNamespace(),
			Labels: map[string]string{
				"agones.dev/gameserver": gs.Name,
			},
			Annotations:     nil,
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "gameserver",
					Port: gameserver.GetGameServerPort(gs).Port,
					TargetPort: intstr.IntOrString{
						IntVal: gameserver.GetGameServerContainerPort(gs),
					},
				},
			},
			Selector: map[string]string{
				"agones.dev/gameserver": gs.Name,
			},
		},
	}

	result, err := r.Client.CoreV1().Services(gs.Namespace).Create(service)
	if err != nil {
		r.logger.WithError(err).Errorf("failed to create service %s", service.Name)
		return nil, errors.Wrap(err, "failed to create service")
	}

	return result, nil
}
