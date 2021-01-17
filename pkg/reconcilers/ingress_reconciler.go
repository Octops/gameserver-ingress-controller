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
	"strconv"
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

	if domain, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressDomain); !ok || len(domain) == 0 {
		return &v1beta1.Ingress{}, errors.Errorf("failed to create ingress, the \"%s\" annotation is either not present or null on the gameserver \"%s\"", gameserver.OctopsAnnotationIngressDomain, gs.Name)
	}

	// TODO: Use octops.io/terminate-tls to define the issuer, octops.io/issuer-tls-name
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: gs.Name,
			Labels: map[string]string{
				"agones.dev/gameserver": gs.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain]),
					},
					SecretName: fmt.Sprintf("%s-tls", gs.Name),
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s", gs.Name, gs.Annotations[gameserver.OctopsAnnotationIngressDomain]),
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

	if err := r.SetTLSIssuer(gs, ingress); err != nil {
		return nil, errors.Wrap(err, "failed to set TLS issuer")
	}

	result, err := r.Client.NetworkingV1beta1().Ingresses(gs.Namespace).Create(ingress)
	if err != nil {
		r.logger.WithError(err).Errorf("failed to create ingress %s", ingress.Name)
		return nil, errors.Wrap(err, "failed to create ingress")
	}

	return result, nil
}

func (r *IngressReconciler) SetTLSIssuer(gs *agonesv1.GameServer, ingress *v1beta1.Ingress) error {
	terminate, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationTerminateTLS)
	if !ok || len(terminate) == 0 {
		return nil
	}

	terminateTLS, err := strconv.ParseBool(terminate)
	if err != nil {
		msgErr := errors.Errorf("annotation %s for %s must be \"true\" or \"false\"", gameserver.OctopsAnnotationTerminateTLS, gs.Name)
		r.logger.Error(msgErr)

		return msgErr
	}

	if !terminateTLS {
		r.logger.Debugf("ignoring TLS setup for %s, %s set to %v", gs.Name, gameserver.OctopsAnnotationTerminateTLS, terminateTLS)
		return nil
	}

	issuer, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIssuerName)
	if !ok || len(issuer) == 0 {
		msgErr := errors.Errorf("annotation %s for %s must be present and not null, check your Fleet or GameServer manifest.", gameserver.OctopsAnnotationIssuerName, gs.Name)
		r.logger.Error(msgErr)

		return msgErr
	}

	ingress.Annotations = map[string]string{
		gameserver.CertManagerAnnotationIssuer: issuer,
	}

	return nil
}
