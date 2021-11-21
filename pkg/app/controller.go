package app

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/controller"
	"github.com/Octops/gameserver-ingress-controller/pkg/handlers"
	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/Octops/gameserver-ingress-controller/pkg/manager"
	"time"
)

type Config struct {
	Kubeconfig string
	SyncPeriod string
	Port       int
	Verbose    bool
}

func StartController(ctx context.Context, config Config) error {
	logger := runtime.NewLogger(config.Verbose)

	clientConf, err := k8sutil.NewClusterConfig(config.Kubeconfig)
	if err != nil {
		logger.Fatal(err)
	}

	duration, err := time.ParseDuration(config.SyncPeriod)
	if err != nil {
		logger.WithError(err).Fatalf("error parsing sync-period flag: %s", config.SyncPeriod)
	}

	mgr, err := manager.NewManager(clientConf, manager.Options{
		SyncPeriod: &duration,
		Port:       config.Port,
	})

	logger.Info("starting gameserver controller")
	ctrl, err := controller.NewGameServerController(mgr, handlers.NewGameSeverEventHandler(clientConf, mgr.GetEventRecorderFor("gameserver-ingress-controller")), controller.Options{
		For: &agonesv1.GameServer{},
	})

	if err := ctrl.Start(ctx); err != nil {
		logger.Fatal(err)
	}

	return nil
}
