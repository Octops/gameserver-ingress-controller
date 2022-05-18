package stores

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	v1 "agones.dev/agones/pkg/client/informers/externalversions/agones/v1"
	"context"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"time"
)

type AgonesStore struct {
	*versioned.Clientset
	v1.GameServerInformer
}

func NewAgonesStore(ctx context.Context, config *rest.Config, resyncPeriod time.Duration) (*AgonesStore, error) {
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create the agones api clientset")
	}

	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, resyncPeriod)
	gameservers := agonesInformerFactory.Agones().V1().GameServers()

	go agonesInformerFactory.Start(ctx.Done())

	store := &AgonesStore{agonesClient, gameservers}

	if err := store.HasSynced(ctx); err != nil {
		return nil, errors.Wrap(err, "Agones failed to sync cache")
	}

	return store, nil
}

func (s *AgonesStore) UpdateGameServer(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	result, err := s.AgonesV1().GameServers(gs.Namespace).Update(ctx, gs, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update gameserver %s", k8sutil.Namespaced(gs))
	}

	return result, nil
}

func (s *AgonesStore) GetGameServer(ctx context.Context, name, namespace string) (*agonesv1.GameServer, error) {
	result, err := s.GameServerInformer.Lister().GameServers(namespace).Get(name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve gameserver %s/%s", namespace, name)
	}

	return result, nil
}

func (s *AgonesStore) HasSynced(ctx context.Context) error {
	gsInformer := s.Informer()

	f := func() error {
		stopper, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()

		runtime.Logger().WithField("component", "store").Info("waiting for Agones cache to sync")
		if !cache.WaitForCacheSync(stopper.Done(), gsInformer.HasSynced) {
			return errors.New("timed out waiting for Agones cache to sync")
		}
		return nil
	}

	return withRetry(time.Second*5, 5, f)
}
