package reconcilers

import (
	"context"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
)

type ServiceStore interface {
	CreateService(ctx context.Context, service *corev1.Service, options metav1.CreateOptions) (*corev1.Service, error)
	GetService(name, namespace string) (*corev1.Service, error)
}

type ServiceReconciler struct {
	store    ServiceStore
	recorder *record.EventRecorder
}

func NewServiceReconciler(store ServiceStore, recorder *record.EventRecorder) *ServiceReconciler {
	return &ServiceReconciler{
		store:    store,
		recorder: recorder,
	}
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, gs *agonesv1.GameServer) (*corev1.Service, error) {
	service, err := r.store.GetService(gs.Name, gs.Namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return r.reconcileNotFound(ctx, gs)
		}

		return &corev1.Service{}, errors.Wrapf(err, "error retrieving Service %s from namespace %s", gs.Name, gs.Namespace)
	}

	//TODO: Validate if details still match the GS info
	return service, nil
}

func (r *ServiceReconciler) reconcileNotFound(ctx context.Context, gs *agonesv1.GameServer) (*corev1.Service, error) {
	r.recorder.RecordCreating(gs, record.ServiceKind)

	opts := []ServiceOption{
		WithCustomServiceAnnotations(),
		WithCustomServiceAnnotationsTemplate(),
	}

	service, err := newService(gs, opts...)
	if err != nil {
		r.recorder.RecordFailed(gs, record.ServiceKind, err)
		return nil, errors.Wrapf(err, "failed to create service for gameserver %s", gs.Name)
	}

	result, err := r.store.CreateService(ctx, service, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			r.recorder.RecordFailed(gs, record.ServiceKind, err)
			return nil, errors.Wrap(err, "failed to create service")
		}
		runtime.Logger().Debug(err)
	}

	r.recorder.RecordSuccess(gs, record.ServiceKind)
	return result, nil
}

func newService(gs *agonesv1.GameServer, options ...ServiceOption) (*corev1.Service, error) {
	ref := metav1.NewControllerRef(gs, agonesv1.SchemeGroupVersion.WithKind("GameServer"))
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gs.Name,
			Namespace: gs.GetNamespace(),
			Labels: map[string]string{
				"agones.dev/gameserver": gs.Name,
			},
			Annotations:     map[string]string{},
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
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

	for _, opt := range options {
		if err := opt(gs, service); err != nil {
			return nil, err
		}
	}

	return service, nil
}
