package controller

import (
	"context"
	"github.com/Octops/gameserver-ingress-controller/pkg/handlers"
	"github.com/Octops/gameserver-ingress-controller/pkg/reconcilers"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Options struct {
	For  client.Object
	Owns client.Object
}

// GameServerController watches for events associated to a particular resource type like GameServers or Fleets.
// It uses the passed EventHandler argument to send back the current state of the world.
type GameServerController struct {
	logger *logrus.Entry
	manager.Manager
}

func NewGameServerController(mgr manager.Manager, eventHandler handlers.EventHandler, options Options) (*GameServerController, error) {
	optFor := reflect.TypeOf(options.For).Elem().String()
	logger := logrus.WithFields(logrus.Fields{
		"source":          "controller",
		"controller_type": optFor,
	})

	err := ctrl.NewControllerManagedBy(mgr).
		For(options.For).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(event event.CreateEvent) bool {
				// Implement some logic here and if returns true if you think that
				// this event should be sent to the reconciler or false otherwise
				return true
			},
			DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
				return true
			},
			UpdateFunc: func(updateEvent event.UpdateEvent) bool {
				return true
			},
			GenericFunc: func(genericEvent event.GenericEvent) bool {
				return true
			},
		}).
		Watches(&source.Kind{Type: options.For}, &handler.Funcs{
			CreateFunc: func(createEvent event.CreateEvent, limitingInterface workqueue.RateLimitingInterface) {
				// OnAdd is triggered only when the controller is syncing its cache.
				// It does not map ot the resource creation event triggered by Kubernetes
				request := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: createEvent.Object.GetNamespace(),
						Name:      createEvent.Object.GetName(),
					},
				}

				//TODO: Investigate if controller require this Done. Keeping doubles the reconcile calls
				//defer limitingInterface.Done(request)

				if err := eventHandler.OnAdd(createEvent.Object); err != nil {
					limitingInterface.AddRateLimited(request)
					return
				}

				limitingInterface.Forget(request)
			},
			UpdateFunc: func(updateEvent event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
				request := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: updateEvent.ObjectNew.GetNamespace(),
						Name:      updateEvent.ObjectNew.GetName(),
					},
				}

				//TODO: Investigate if controller require this Done. Keeping doubles the reconcile calls
				//defer limitingInterface.Done(request)

				if err := eventHandler.OnUpdate(updateEvent.ObjectOld, updateEvent.ObjectNew); err != nil {
					limitingInterface.AddRateLimited(request)
					return
				}

				limitingInterface.Forget(request)
			},
			DeleteFunc: func(deleteEvent event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {

				request := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: deleteEvent.Object.GetNamespace(),
						Name:      deleteEvent.Object.GetName(),
					},
				}

				if err := eventHandler.OnDelete(deleteEvent.Object); err != nil {
					limitingInterface.AddRateLimited(request)
					return
				}

				limitingInterface.Forget(request)
			},
		}).
		Complete(&reconcilers.Reconciler{
			Obj:    options.For,
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})

	if err != nil {
		return nil, err
	}

	controller := &GameServerController{
		logger:  logger,
		Manager: mgr,
	}

	logger.Infof("controller created for resource of type %s", optFor)
	return controller, nil
}

func (c *GameServerController) Start(ctx context.Context) error {

	go func() {
		chDone := ctrl.SetupSignalHandler()
		c.Manager.Start(chDone)
	}()

	<-ctx.Done()

	return nil
}
