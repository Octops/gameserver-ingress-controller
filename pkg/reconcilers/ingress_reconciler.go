package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"fmt"
	. "github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

var (
	defaultPathType = networkingv1.PathTypePrefix
)

type IngressReconciler struct {
	logger   *logrus.Entry
	recorder record.EventRecorder
	Client   *kubernetes.Clientset
}

func NewIngressReconciler(client *kubernetes.Clientset, recorder record.EventRecorder) *IngressReconciler {
	return &IngressReconciler{
		logger:   Logger().WithField("role", "ingress_reconciler"),
		recorder: recorder,
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
	r.RecordCreating(gs)

	mode := gameserver.GetIngressRoutingMode(gs)
	issuer := gameserver.GetTLSCertIssuer(gs)

	opts := []IngressOption{
		WithIngressRule(mode),
		WithTLS(mode),
		WithTLSCertIssuer(issuer),
	}

	ingress, err := newIngress(gs, opts...)
	if err != nil {
		r.RecordFailed(gs, err)
		return nil, errors.Wrapf(err, "failed to create ingress for gameserver %s", gs.Name)
	}

	result, err := r.Client.NetworkingV1().Ingresses(gs.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		r.RecordFailed(gs, err)
		return nil, errors.Wrapf(err, "failed to push ingress %s for gameserver %s", ingress.Name, gs.Name)
	}

	r.RecordSuccess(gs)
	return result, nil
}

func (r *IngressReconciler) RecordFailed(gs *agonesv1.GameServer, err error) {
	r.recordEvent(gs, EventTypeWarning, ReasonReconcileFailed, fmt.Sprintf("Failed to create ingress for gameserver %s/%s: %s", gs.Namespace, gs.Name, err))
}

func (r *IngressReconciler) RecordSuccess(gs *agonesv1.GameServer) {
	r.recordEvent(gs, EventTypeNormal, ReasonReconciled, fmt.Sprintf("Ingress created for gameserver %s/%s", gs.Namespace, gs.Name))
}

func (r *IngressReconciler) RecordCreating(gs *agonesv1.GameServer) {
	r.recordEvent(gs, EventTypeNormal, ReasonReconcileCreating, fmt.Sprintf("Creating Ingress for gameserver %s/%s", gs.Namespace, gs.Name))
}

func (r *IngressReconciler) recordEvent(object runtime.Object, eventtype, reason, message string) {
	r.recorder.Event(object, eventtype, reason, message)
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
