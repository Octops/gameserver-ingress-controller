package stores

import (
	"context"
	"time"

	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/pkg/errors"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
)

type Store struct {
	*serviceStore
	*ingressStore
	*gatewayStore
}

func NewStore(ctx context.Context, client kubernetes.Interface, restConfig *rest.Config, gatewayEnabled bool) (*Store, error) {
	factory := informers.NewSharedInformerFactory(client, 0)
	services := factory.Core().V1().Services()
	ingresses := factory.Networking().V1().Ingresses()

	go factory.Start(ctx.Done())

	store := &Store{
		serviceStore: newServiceStore(client, services),
		ingressStore: newIngressStore(client, ingresses),
	}

	if gatewayEnabled {
		gwClient, err := gatewayclient.NewForConfig(restConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create gateway-api client")
		}
		gwFactory := gatewayinformers.NewSharedInformerFactory(gwClient, 0)
		httpRoutes := gwFactory.Gateway().V1().HTTPRoutes()
		go gwFactory.Start(ctx.Done())
		store.gatewayStore = newGatewayStore(gwClient, httpRoutes)
	}

	if err := store.HasSynced(ctx); err != nil {
		return nil, errors.Wrap(err, "store failed to sync K8S cache")
	}

	return store, nil
}

func (s *Store) HasSynced(ctx context.Context) error {
	syncFuncs := []cache.InformerSynced{
		s.serviceStore.informer.Informer().HasSynced,
		s.ingressStore.informer.Informer().HasSynced,
	}
	if s.gatewayStore != nil {
		syncFuncs = append(syncFuncs, s.gatewayStore.informer.Informer().HasSynced)
	}

	f := func() error {
		stopper, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()

		runtime.Logger().WithField("component", "store").Info("waiting for K8S cache to sync")
		if !cache.WaitForCacheSync(stopper.Done(), syncFuncs...) {
			return errors.New("timed out waiting for K8S cache to sync")
		}
		return nil
	}

	return withRetry(time.Second*5, 5, f)
}

// withRetry will wait for the interval before calling the f function for a max number of retries.
func withRetry(interval time.Duration, maxRetries int, f func() error) error {
	var err error
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		time.Sleep(interval)
		if err = f(); err == nil {
			return nil
		}
		continue
	}

	return errors.Wrapf(err, "retry failed after %d attempts", maxRetries)
}
