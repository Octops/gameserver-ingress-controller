package app

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/pkg/controller"
	"github.com/Octops/gameserver-ingress-controller/pkg/handlers"
	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/Octops/gameserver-ingress-controller/pkg/manager"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
	"github.com/Octops/gameserver-ingress-controller/pkg/stores"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type Config struct {
	Kubeconfig             string
	SyncPeriod             string
	Port                   int
	Verbose                bool
	HealthProbeBindAddress string
	MetricsBindAddress     string
}

func StartController(ctx context.Context, logger *logrus.Entry, config Config) error {
	duration, err := time.ParseDuration(config.SyncPeriod)
	if err != nil {
		withFatal(logger, err, fmt.Sprintf("error parsing sync-period flag: %s", config.SyncPeriod))
	}

	mgr, err := manager.NewManager(config.Kubeconfig, manager.Options{
		SyncPeriod:             &duration,
		Port:                   config.Port,
		HealthProbeBindAddress: config.HealthProbeBindAddress,
		MetricsBindAddress:     config.MetricsBindAddress,
	})
	if err != nil {
		withFatal(logger, err, "failed to create controller manager")
	}

	client, err := k8sutil.NewClientSet(config.Kubeconfig)
	if err != nil {
		withFatal(logger, err, "failed to create kubernetes client")
	}

	store, err := stores.NewStore(ctx, client)
	if err != nil {
		withFatal(logger, err, "failed to create store")
	}

	logger.WithField("component", "controller").Info("starting gameserver controller")

	recorder := mgr.GetEventRecorderFor("gameserver-ingress-controller")
	handler := handlers.NewGameSeverEventHandler(store, record.NewEventRecorder(recorder))
	ctrl, err := controller.NewGameServerController(ctx, mgr, handler, controller.Options{
		For: &agonesv1.GameServer{},
	})

	if err != nil {
		withFatal(logger, err, "failed to create controller")
	}

	if err := ctrl.Start(ctx); err != nil {
		withFatal(logger, err, "failed to start controller")
	}

	return nil
}

func withFatal(logger *logrus.Entry, err error, msg string) {
	logger.Fatal(errors.Wrap(err, msg))
}
