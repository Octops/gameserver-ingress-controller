package manager

import (
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
)

type Options struct {
	SyncPeriod              *time.Duration
	Port                    int
	HealthProbeBindAddress  string
	MetricsBindAddress      string
	MaxConcurrentReconciles int
}

type Manager struct {
	manager.Manager
}

func NewManager(kubeconfig string, options Options) (*Manager, error) {
	c, err := k8sutil.NewClusterConfig(kubeconfig)
	if err != nil {
		return nil, withError(errors.Wrap(err, "failed to create cluster config"))
	}

	if options.MaxConcurrentReconciles <= 0 {
		options.MaxConcurrentReconciles = 1
	}

	mgr, err := manager.New(c, manager.Options{
		Cache: cache.Options{
			SyncPeriod: options.SyncPeriod,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: options.Port,
		}),
		Controller: config.Controller{
			MaxConcurrentReconciles: options.MaxConcurrentReconciles,
		},
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
