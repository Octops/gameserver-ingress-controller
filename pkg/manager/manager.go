package manager

import (
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
)

type Options struct {
	SyncPeriod             *time.Duration
	Port                   int
	HealthProbeBindAddress string
	MetricsBindAddress     string
}

type Manager struct {
	manager.Manager
}

func NewManager(kubeconfig string, options Options) (*Manager, error) {
	config, err := k8sutil.NewClusterConfig(kubeconfig)
	if err != nil {
		return nil, withError(errors.Wrap(err, "failed to create cluster config"))
	}

	mgr, err := manager.New(config, manager.Options{
		SyncPeriod:             options.SyncPeriod,
		Port:                   options.Port,
		MetricsBindAddress:     options.MetricsBindAddress,
		HealthProbeBindAddress: options.HealthProbeBindAddress,
	})
	if err != nil {
		return nil, withError(err)
	}

	if err := mgr.AddHealthzCheck("/", healthz.Ping); err != nil {
		return nil, withError(err)
	}

	if err := mgr.AddReadyzCheck("/", healthz.Ping); err != nil {
		return nil, withError(err)
	}

	return &Manager{mgr}, nil
}

func withError(err error) error {
	return errors.Wrap(err, "failed to create manager")
}
