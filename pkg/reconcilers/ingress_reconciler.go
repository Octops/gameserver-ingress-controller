package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressStore interface {
	CreateIngress(ctx context.Context, ingress *networkingv1.Ingress, options metav1.CreateOptions) (*networkingv1.Ingress, error)
	GetIngress(name, namespace string) (*networkingv1.Ingress, error)
}

type IngressReconciler struct {
	store    IngressStore
	recorder *record.EventRecorder
}

func NewIngressReconciler(store IngressStore, recorder *record.EventRecorder) *IngressReconciler {
	return &IngressReconciler{
		store:    store,
		recorder: recorder,
	}
}

func (r *IngressReconciler) Reconcile(ctx context.Context, gs *agonesv1.GameServer) (*networkingv1.Ingress, error) {
	ingress, err := r.store.GetIngress(gs.Name, gs.Namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return r.reconcileNotFound(ctx, gs)
		}

		return nil, errors.Wrapf(err, "error retrieving Ingress %s from namespace %s", gs.Name, gs.Namespace)
	}

	//TODO: Validate if details still match the GS info
	return ingress, nil
}

func (r *IngressReconciler) reconcileNotFound(ctx context.Context, gs *agonesv1.GameServer) (*networkingv1.Ingress, error) {
	r.recorder.RecordCreating(gs, record.IngressKind)

	mode := gameserver.GetIngressRoutingMode(gs)
	issuer := gameserver.GetTLSCertIssuer(gs)

	opts := []IngressOption{
		WithCustomAnnotations(),
		WithCustomAnnotationsTemplate(),
		WithIngressRule(mode),
		WithTLS(mode),
		WithTLSCertIssuer(issuer),
	}

	ingress, err := newIngress(gs, opts...)
	if err != nil {
		r.recorder.RecordFailed(gs, record.IngressKind, err)
		return nil, errors.Wrapf(err, "failed to create ingress for gameserver %s", gs.Name)
	}

	result, err := r.store.CreateIngress(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			r.recorder.RecordFailed(gs, record.IngressKind, err)
			return nil, errors.Wrapf(err, "failed to push ingress %s for gameserver %s", ingress.Name, gs.Name)
		}
		runtime.Logger().Debug(err)
	}

	r.recorder.RecordSuccess(gs, record.IngressKind)
	return result, nil
}

func newIngress(gs *agonesv1.GameServer, options ...IngressOption) (*networkingv1.Ingress, error) {
	if gs == nil {
		return nil, errors.New("gameserver can't be nil")
	}

	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
	ig := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gs.Name,
			Namespace: gs.Namespace,
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
