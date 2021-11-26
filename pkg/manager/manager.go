package manager

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
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

func NewManager(config *rest.Config, options Options) (*Manager, error) {
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
