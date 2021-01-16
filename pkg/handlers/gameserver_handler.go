package handlers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/reconcilers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	OctopsIngressControllerAnnotation = "octops.io/gameserver-ingress-domain"
)

type GameSeverEventHandler struct {
	logger            *logrus.Entry
	client            *kubernetes.Clientset
	serviceReconciler *reconcilers.ServiceReconciler
	ingressReconciler *reconcilers.IngressReconciler
}

func NewGameSeverEventHandler(config *rest.Config) *GameSeverEventHandler {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		runtime.Logger().WithError(err).Fatal("failed to create kubernetes client")
	}

	return &GameSeverEventHandler{
		logger:            runtime.Logger().WithField("role", "event_handler"),
		client:            client,
		serviceReconciler: reconcilers.NewServiceReconciler(client),
		ingressReconciler: reconcilers.NewIngressReconciler(client),
	}
}

func (h *GameSeverEventHandler) OnAdd(obj interface{}) error {
	h.logger.Infof("ADD: %s", obj.(*agonesv1.GameServer).Name)

	// x Check Annotations
	// x Create Service with label selector
	// Get the domain name from the annotation
	// Create certificate
	// Create Ingress with Host gamename.domain.com
	// x Set Owner Reference

	gs := gameserver.FromObject(obj)

	return h.Reconcile(gs)
}

func (h *GameSeverEventHandler) OnUpdate(oldObj interface{}, newObj interface{}) error {
	h.logger.Infof("UPDATE: %s", newObj.(*agonesv1.GameServer).Name)

	gs := gameserver.FromObject(newObj)

	return h.Reconcile(gs)
}

func (h *GameSeverEventHandler) OnDelete(obj interface{}) error {
	h.logger.Infof("DELETE: %s", obj.(*agonesv1.GameServer).Name)

	return nil
}

func (h GameSeverEventHandler) Client() *kubernetes.Clientset {
	return h.client
}

func (h *GameSeverEventHandler) Reconcile(gs *agonesv1.GameServer) error {
	if _, ok := gameserver.HasReconcileAnnotation(gs, OctopsIngressControllerAnnotation); !ok {
		return nil
	}

	if gameserver.IsReady(gs) == false {
		msg := fmt.Sprintf("gameserver %s/%s not ready", gs.Namespace, gs.Name)
		h.logger.Debug(msg)

		return errors.New(msg)
	}

	_, err := h.serviceReconciler.Reconcile(gs)
	if err != nil {
		return errors.Wrap(err, "failed to reconcile gameserver/service")
	}

	_, err = h.ingressReconciler.Reconcile(gs)
	if err != nil {
		return errors.Wrap(err, "failed to reconcile gameserver/ingress")
	}

	return nil
}
