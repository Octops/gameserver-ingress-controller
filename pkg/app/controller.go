package app

import (
	"context"
	"fmt"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/controller"
	"github.com/Octops/gameserver-ingress-controller/pkg/handlers"
	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/Octops/gameserver-ingress-controller/pkg/manager"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
	"github.com/Octops/gameserver-ingress-controller/pkg/stores"
)

type Config struct {
	Kubeconfig              string
	SyncPeriod              string
	Port                    int
	Verbose                 bool
	HealthProbeBindAddress  string
	MetricsBindAddress      string
	MaxConcurrentReconciles int
	// EnableGatewayAPI controls the Gateway API backend.
	// "auto" (default): enable if CRDs are present, warn and disable if not.
	// "true": always enable, fail hard at startup if CRDs are missing.
	// "false": always disable, no informer or client created.
	EnableGatewayAPI string
}

func StartController(ctx context.Context, logger *logrus.Entry, config Config) error {
	duration, err := time.ParseDuration(config.SyncPeriod)
	if err != nil {
		withFatal(logger, err, fmt.Sprintf("error parsing sync-period flag: %s", config.SyncPeriod))
	}

	mgr, err := manager.NewManager(config.Kubeconfig, manager.Options{
		SyncPeriod:              &duration,
		Port:                    config.Port,
		HealthProbeBindAddress:  config.HealthProbeBindAddress,
		MetricsBindAddress:      config.MetricsBindAddress,
		MaxConcurrentReconciles: config.MaxConcurrentReconciles,
	})
	if err != nil {
		withFatal(logger, err, "failed to create controller manager")
	}

	clusterConfig, err := k8sutil.NewClusterConfig(config.Kubeconfig)
	if err != nil {
		withFatal(logger, err, "failed to create cluster config")
	}

	client, err := k8sutil.NewClientSet(config.Kubeconfig)
	if err != nil {
		withFatal(logger, err, "failed to create kubernetes client")
	}

	gatewayEnabled, err := resolveGatewayAPIEnabled(config.EnableGatewayAPI, client, logger)
	if err != nil {
		withFatal(logger, err, "failed to resolve --enable-gateway-api")
	}

	store, err := stores.NewStore(ctx, client, clusterConfig, gatewayEnabled)
	if err != nil {
		withFatal(logger, err, "failed to create store")
	}

	agones, err := stores.NewAgonesStore(ctx, clusterConfig, duration)

	recorder := mgr.GetEventRecorderFor("octops-gameserver-controller")
	handler := handlers.NewGameSeverEventHandler(store, agones, record.NewEventRecorder(recorder), gatewayEnabled)

	ctrl, err := controller.NewGameServerController(ctx, mgr, handler, controller.Options{
		For: &agonesv1.GameServer{},
	})

	if err != nil {
		withFatal(logger, err, "failed to create controller")
	}

	logger.WithField("component", "controller").Info("starting gameserver controller")
	if err := ctrl.Start(ctx); err != nil {
		withFatal(logger, err, "failed to start controller")
	}

	return nil
}

// resolveGatewayAPIEnabled determines whether the Gateway API backend should be
// enabled based on the --enable-gateway-api flag value:
//
//	"false" – always disabled
//	"true"  – always enabled; return error if CRDs are absent
//	"auto"  – enabled if gateway.networking.k8s.io/v1 is registered, disabled otherwise
func resolveGatewayAPIEnabled(mode string, client kubernetes.Interface, logger *logrus.Entry) (bool, error) {
	log := runtime.Logger().WithField("component", "gateway-api")

	switch mode {
	case "false":
		log.Info("Gateway API backend disabled (--enable-gateway-api=false)")
		return false, nil
	case "true":
		if err := checkGatewayAPICRDs(client); err != nil {
			return false, errors.Wrap(err, "Gateway API CRDs not found (--enable-gateway-api=true requires them to be installed)")
		}
		log.Info("Gateway API backend enabled (--enable-gateway-api=true)")
		return true, nil
	default: // "auto"
		if err := checkGatewayAPICRDs(client); err != nil {
			log.Warn("Gateway API CRDs not found — gateway backend disabled. Install CRDs or set --enable-gateway-api=true to require them.")
			return false, nil
		}
		log.Info("Gateway API CRDs detected — gateway backend enabled (--enable-gateway-api=auto)")
		return true, nil
	}
}

// checkGatewayAPICRDs returns nil if the HTTPRoute resource is served under
// gateway.networking.k8s.io/v1. Checking the group alone is insufficient
// because other Gateway API CRDs (Gateway, GatewayClass) may be present while
// HTTPRoute is absent.
func checkGatewayAPICRDs(client kubernetes.Interface) error {
	resources, err := client.Discovery().ServerResourcesForGroupVersion("gateway.networking.k8s.io/v1")
	if err != nil {
		return err
	}
	for _, r := range resources.APIResources {
		if r.Name == "httproutes" {
			return nil
		}
	}
	return errors.New("httproutes not found in gateway.networking.k8s.io/v1")
}

func withFatal(logger *logrus.Entry, err error, msg string) {
	logger.Fatal(errors.Wrap(err, msg))
}
