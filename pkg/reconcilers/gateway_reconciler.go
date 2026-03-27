package reconcilers

import (
	"context"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type HTTPRouteStore interface {
	CreateHTTPRoute(ctx context.Context, route *gatewayv1.HTTPRoute, options metav1.CreateOptions) (*gatewayv1.HTTPRoute, error)
	GetHTTPRoute(name, namespace string) (*gatewayv1.HTTPRoute, error)
}

type GatewayReconciler struct {
	store    HTTPRouteStore
	recorder *record.EventRecorder
}

func NewGatewayReconciler(store HTTPRouteStore, recorder *record.EventRecorder) *GatewayReconciler {
	return &GatewayReconciler{
		store:    store,
		recorder: recorder,
	}
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, gs *agonesv1.GameServer) (*gatewayv1.HTTPRoute, bool, error) {
	route, err := r.store.GetHTTPRoute(gs.Name, gs.Namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return r.reconcileNotFound(ctx, gs)
		}

		return nil, false, errors.Wrapf(err, "error retrieving HTTPRoute %s from namespace %s", gs.Name, gs.Namespace)
	}

	return route, false, nil
}

func (r *GatewayReconciler) reconcileNotFound(ctx context.Context, gs *agonesv1.GameServer) (*gatewayv1.HTTPRoute, bool, error) {
	r.recorder.RecordCreating(gs, record.HTTPRouteKind)

	if _, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationTerminateTLS); ok {
		r.recorder.RecordWarning(gs, record.HTTPRouteKind, "annotation octops.io/terminate-tls has no effect in gateway mode — configure TLS on the Gateway listener instead")
	}
	if _, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIssuerName); ok {
		r.recorder.RecordWarning(gs, record.HTTPRouteKind, "annotation octops.io/issuer-tls-name has no effect in gateway mode — use a cert-manager Certificate resource linked to the Gateway listener instead")
	}

	mode := gameserver.GetIngressRoutingMode(gs)

	opts := []HTTPRouteOption{
		WithCustomHTTPRouteAnnotations(),
		WithCustomHTTPRouteAnnotationsTemplate(),
		WithHTTPRouteParentRef(),
		WithHTTPRouteRules(mode),
	}

	route, err := newHTTPRoute(gs, opts...)
	if err != nil {
		r.recorder.RecordFailed(gs, record.HTTPRouteKind, err)
		return nil, false, errors.Wrapf(err, "failed to create HTTPRoute for gameserver %s", gs.Name)
	}

	result, err := r.store.CreateHTTPRoute(ctx, route, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			r.recorder.RecordFailed(gs, record.HTTPRouteKind, err)
			return nil, false, errors.Wrapf(err, "failed to push HTTPRoute %s for gameserver %s", route.Name, gs.Name)
		}
		runtime.Logger().Debug(err)
	}

	r.recorder.RecordSuccess(gs, record.HTTPRouteKind)
	return result, true, nil
}

func newHTTPRoute(gs *agonesv1.GameServer, options ...HTTPRouteOption) (*gatewayv1.HTTPRoute, error) {
	if gs == nil {
		return nil, errors.New("gameserver can't be nil")
	}

	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
	route := &gatewayv1.HTTPRoute{
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
		if err := opt(gs, route); err != nil {
			return nil, err
		}
	}

	return route, nil
}
