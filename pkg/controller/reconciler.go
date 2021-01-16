package controller

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler handles events when resources are reconciled. The interval is configured on the Manager's level.
type Reconciler struct {
	logger *logrus.Entry
	obj    runtime.Object
	client.Client
	scheme *runtime.Scheme
}

// Warning: This method is possible not meant to be used. It has a particular use case but the broadcaster uses a shorter
// Sync period that triggers OnUpdate events. Right now this Reconcile function is useless for the broadcaster.
// It should be explored in the future.

// TODO: Evaluate is Reconcile should be made an argument for the Controller. Reconcile can be used for general uses cases
// where control over very specific events matter. Right now it is just a STDOUT output.
// Reconcile is called on every reconcile event. It does not differ between add, update, delete.
// Its function is purely informative and events are handled back to the broadcaster specific event handlers.
func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	//ctx := context.Background()
	//obj := r.obj.DeepCopyObject()
	//if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
	//	if apierrors.IsNotFound(err) {
	//		r.logger.WithField("type", reflect.TypeOf(obj).String()).Debugf("resource \"%s\" not found", req.NamespacedName)
	//		return ctrl.Result{}, nil
	//	}
	//
	//	r.logger.WithError(err).Error()
	//
	//	return reconcile.Result{}, err
	//}

	//r.logger.Debugf("OnReconcile: %s (%s)", req.NamespacedName, reflect.TypeOf(obj).String())

	return reconcile.Result{}, nil
}
