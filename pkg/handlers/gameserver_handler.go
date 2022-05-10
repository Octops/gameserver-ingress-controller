package handlers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/reconcilers"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
	"github.com/Octops/gameserver-ingress-controller/pkg/stores"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type GameSeverEventHandler struct {
	logger            *logrus.Entry
	client            *kubernetes.Clientset
	serviceReconciler *reconcilers.ServiceReconciler
	ingressReconciler *reconcilers.IngressReconciler
}

func NewGameSeverEventHandler(store *stores.Store, recorder *record.EventRecorder) *GameSeverEventHandler {
	return &GameSeverEventHandler{
		logger:            runtime.Logger().WithField("component", "event_handler"),
		serviceReconciler: reconcilers.NewServiceReconciler(store, recorder),
		ingressReconciler: reconcilers.NewIngressReconciler(store, recorder),
	}
}

func (h *GameSeverEventHandler) OnAdd(obj interface{}) error {
	gs := gameserver.FromObject(obj)

	if err := h.Reconcile(h.logger.WithField("event", "added"), gs); err != nil {
		h.logger.Error(err)
	}

	return nil
}

func (h *GameSeverEventHandler) OnUpdate(_ interface{}, newObj interface{}) error {
	gs := gameserver.FromObject(newObj)

	if err := h.Reconcile(h.logger.WithField("event", "updated"), gs); err != nil {
		h.logger.Error(err)
	}

	return nil
}

func (h *GameSeverEventHandler) OnDelete(obj interface{}) error {
	gs := obj.(*agonesv1.GameServer)
	h.logger.WithField("event", "deleted").Infof("%s/%s", gs.Namespace, gs.Name)

	return nil
}

func (h *GameSeverEventHandler) Reconcile(logger *logrus.Entry, gs *agonesv1.GameServer) error {
	if _, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressMode); !ok {
		logger.Infof("skipping %s/%s, annotation %s not present", gs.Namespace, gs.Name, gameserver.OctopsAnnotationIngressMode)
		return nil
	}

	//If a game server is in a Shutdown state it will not trigger reconcile
	if gameserver.IsShutdown(gs) {
		logger.WithField("event", "shutdown").Infof("%s/%s", gs.Namespace, gs.Name)

		return nil
	}

	//Only Scheduled, ReadyState and Ready game server states will trigger reconcile
	if gameserver.MustReconcile(gs) == false {
		msg := fmt.Sprintf("%s/%s/%s not reconciled, waiting for Scheduled, ReadyState or Ready state", gs.Namespace, gs.Name, gs.Status.State)
		logger.Info(msg)

		return nil
	}

	ctx := context.TODO()
	_, err := h.serviceReconciler.Reconcile(ctx, gs)
	if err != nil {
		return errors.Wrapf(err, "failed to reconcile service %s/%s", gs.Namespace, gs.Name)
	}

	_, err = h.ingressReconciler.Reconcile(ctx, gs)
	if err != nil {
		return errors.Wrapf(err, "failed to reconcile ingress %s/%s", gs.Namespace, gs.Name)
	}

	msg := fmt.Sprintf("%s/%s/%s reconciled", gs.Namespace, gs.Name, gs.Status.State)
	logger.Info(msg)

	return nil
}
