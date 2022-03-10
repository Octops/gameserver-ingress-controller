package app

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/controller"
	"github.com/Octops/gameserver-ingress-controller/pkg/handlers"
	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/Octops/gameserver-ingress-controller/pkg/manager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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

func StartController(ctx context.Context, config Config) error {
	logger := runtime.NewLogger(config.Verbose)

	clientConf, err := k8sutil.NewClusterConfig(config.Kubeconfig)
	if err != nil {
		withFatal(logger, err, "failed to create cluster config")
	}

	duration, err := time.ParseDuration(config.SyncPeriod)
	if err != nil {
		withFatal(logger, err, fmt.Sprintf("error parsing sync-period flag: %s", config.SyncPeriod))
	}

	mgr, err := manager.NewManager(clientConf, manager.Options{
		SyncPeriod:             &duration,
		Port:                   config.Port,
		HealthProbeBindAddress: config.HealthProbeBindAddress,
		MetricsBindAddress:     config.MetricsBindAddress,
	})
	if err != nil {
		withFatal(logger, err, "failed to create controller manager")
	}

	client, err := kubernetes.NewForConfig(clientConf)
	if err != nil {
		withFatal(logger, err, "failed to create kubernetes client")
	}

	factory := informers.NewSharedInformerFactory(client, 0)
	services := factory.Core().V1().Services()
	ingresses := factory.Networking().V1().Ingresses()
	svcInformer := services.Informer()
	ingInformer := ingresses.Informer()

	stopper := make(chan struct{})
	defer close(stopper)
	go factory.Start(stopper)

	logger.Info("waiting for cache to sync")
	if !cache.WaitForCacheSync(stopper, svcInformer.HasSynced, ingInformer.HasSynced) {
		withFatal(logger, fmt.Errorf("timed out waiting for caches to sync"), "failed to sync cache")
	}

	logger.Info("starting gameserver controller")
	handler := handlers.NewGameSeverEventHandler(clientConf, services, ingresses, mgr.GetEventRecorderFor("gameserver-ingress-controller"))
	ctrl, err := controller.NewGameServerController(mgr, handler, controller.Options{
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
