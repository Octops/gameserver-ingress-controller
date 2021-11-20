package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type IngressReconciler struct {
	recorder *EventRecorder
	Client   *kubernetes.Clientset
}

func NewIngressReconciler(client *kubernetes.Clientset, recorder record.EventRecorder) *IngressReconciler {
	return &IngressReconciler{
		recorder: NewEventRecorder(recorder),
		Client:   client,
	}
}

func (r *IngressReconciler) Reconcile(ctx context.Context, gs *agonesv1.GameServer) (*networkingv1.Ingress, error) {
	ingress, err := r.Client.NetworkingV1().Ingresses(gs.Namespace).Get(ctx, gs.Name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return &networkingv1.Ingress{}, errors.Wrapf(err, "error retrieving Ingress %s from namespace %s", gs.Name, gs.Namespace)
		}

		return r.reconcileNotFound(ctx, gs)
	}

	//TODO: Validate if details still match the GS info
	return ingress, nil
}

func (r *IngressReconciler) reconcileNotFound(ctx context.Context, gs *agonesv1.GameServer) (*networkingv1.Ingress, error) {
	r.recorder.RecordCreating(gs, IngressKind)

	mode := gameserver.GetIngressRoutingMode(gs)
	issuer := gameserver.GetTLSCertIssuer(gs)

	opts := []IngressOption{
		WithCustomAnnotations(),
		WithIngressRule(mode),
		WithTLS(mode),
		WithTLSCertIssuer(issuer),
	}

	ingress, err := newIngress(gs, opts...)
	if err != nil {
		r.recorder.RecordFailed(gs, IngressKind, err)
		return nil, errors.Wrapf(err, "failed to create ingress for gameserver %s", gs.Name)
	}

	result, err := r.Client.NetworkingV1().Ingresses(gs.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		r.recorder.RecordFailed(gs, IngressKind, err)
		return nil, errors.Wrapf(err, "failed to push ingress %s for gameserver %s", ingress.Name, gs.Name)
	}

	r.recorder.RecordSuccess(gs, IngressKind)
	return result, nil
}

func newIngress(gs *agonesv1.GameServer, options ...IngressOption) (*networkingv1.Ingress, error) {
	if gs == nil {
		return nil, errors.New("gameserver can't be nil")
	}

	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
	ig := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: gs.Name,
			Labels: map[string]string{
				gameserver.AgonesGameServerNameLabel: gs.Name,
			},
			Annotations:     map[string]string{},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
	}

	for _, opt := range options {
		if err := opt(gs, ig); err != nil {
			return nil, err
		}
	}

	return ig, nil
}
