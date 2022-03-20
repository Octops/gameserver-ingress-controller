package stores

import (
	"context"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/pkg/errors"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

type Store struct {
	*serviceStore
	*ingressStore
}

func NewStore(ctx context.Context, client kubernetes.Interface) (*Store, error) {
	factory := informers.NewSharedInformerFactory(client, 0)
	services := factory.Core().V1().Services()
	ingresses := factory.Networking().V1().Ingresses()

	go factory.Start(ctx.Done())

	store := &Store{
		newServiceStore(client, services),
		newIngressStore(client, ingresses),
	}

	if err := store.HasSynced(ctx); err != nil {
		return nil, errors.Wrap(err, "store failed to sync cache")
	}

	return store, nil
}

func (s *Store) HasSynced(ctx context.Context) error {
	svcInformer := s.serviceStore.informer.Informer()
	ingInformer := s.ingressStore.informer.Informer()

	f := func() error {
		stopper, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()

		runtime.Logger().WithField("component", "store").Info("waiting for cache to sync")
		if !cache.WaitForCacheSync(stopper.Done(), svcInformer.HasSynced, ingInformer.HasSynced) {
			return errors.New("timed out waiting for caches to sync")
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
