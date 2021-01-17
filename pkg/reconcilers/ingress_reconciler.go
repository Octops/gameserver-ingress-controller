package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type IngressReconciler struct {
	logger *logrus.Entry
	Client *kubernetes.Clientset
}

func NewIngressReconciler(client *kubernetes.Clientset) *IngressReconciler {
	return &IngressReconciler{
		logger: runtime.Logger().WithField("role", "ingress_reconciler"),
		Client: client,
	}
}

func (r IngressReconciler) Reconcile(gs *agonesv1.GameServer) (*v1beta1.Ingress, error) {
	ingress, err := r.Client.NetworkingV1beta1().Ingresses(gs.Namespace).Get(gs.Name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return &v1beta1.Ingress{}, errors.Wrapf(err, "error retrieving Ingress %s from namespace %s", gs.Name, gs.Namespace)
		}

		return r.reconcileNotFound(gs)
	}

	//TODO: Validate if details still match the GS info

	return ingress, nil
}

func (r *IngressReconciler) reconcileNotFound(gs *agonesv1.GameServer) (*v1beta1.Ingress, error) {
	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))

	if domain, ok := gameserver.HasAnnotation(gs, gameserver.DomainAnnotation); !ok || len(domain) == 0 {
		return &v1beta1.Ingress{}, errors.Errorf("failed to create ingress, the \"%s\" annotation is either not present or null on the gameserver \"%s\"", gameserver.DomainAnnotation, gs.Name)
	}

	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: gs.Name,
			Labels: map[string]string{
				"agones.dev/gameserver": gs.Name,
			},
			Annotations: map[string]string{
				//"cert-manager.io/issuer": "letsencrypt-staging",
				//"cert-manager.io/issuer": "letsencrypt-prod",
				"cert-manager.io/issuer": "selfsigned-issuer",
			},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.DomainAnnotation]),
					},
					SecretName: fmt.Sprintf("%s-tls", gs.Name),
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.DomainAnnotation]),
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: gs.Name,
										ServicePort: intstr.IntOrString{
											IntVal: gameserver.GetGameServerPort(gs).Port,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := r.Client.NetworkingV1beta1().Ingresses(gs.Namespace).Create(ingress)
	if err != nil {
		r.logger.WithError(err).Errorf("failed to create ingress %s", ingress.Name)
		return nil, errors.Wrap(err, "failed to create ingress")
	}

	return result, nil
}
