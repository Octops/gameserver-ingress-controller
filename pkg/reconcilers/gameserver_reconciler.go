package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/k8sutil"
	"github.com/Octops/gameserver-ingress-controller/pkg/record"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/retry"
	"strconv"
	"strings"
)

type GameServerStore interface {
	UpdateGameServer(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error)
	GetGameServer(ctx context.Context, name, namespace string) (*agonesv1.GameServer, error)
}

type GameServerReconciler struct {
	store    GameServerStore
	recorder *record.EventRecorder
}

func NewGameServerReconciler(store GameServerStore, recorder *record.EventRecorder) *GameServerReconciler {
	return &GameServerReconciler{store: store, recorder: recorder}
}

func (r *GameServerReconciler) Reconcile(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	must, err := r.MustReconcile(gs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile gameserver %s/%s", gs.Namespace, gs.Name)
	}

	if must == false {
		return gs, nil
	}

	return r.reconcile(ctx, gs)
}

func (r *GameServerReconciler) reconcile(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	result := &agonesv1.GameServer{}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		g, err := r.store.GetGameServer(ctx, gs.Name, gs.Namespace)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve gameserver %s", k8sutil.Namespaced(gs))
		}

		deepCopy := g.DeepCopy()
		deepCopy.Annotations[gameserver.OctopsAnnotationGameServerIngressReady] = "true"

		result, err = r.store.UpdateGameServer(ctx, deepCopy)
		if err != nil {
			return errors.Wrapf(err, "failed to update gameserver %s", k8sutil.Namespaced(deepCopy))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	r.recorder.RecordEvent(result, fmt.Sprintf("GameServer annotated with %s", gameserver.OctopsAnnotationGameServerIngressReady))
	r.recordDeprecatedAnnotations(result)
	return result, nil
}

func (r *GameServerReconciler) recordDeprecatedAnnotations(gs *agonesv1.GameServer) {
	if _, ok := gs.Annotations[gameserver.OctopsAnnotationIngressClassNameLegacy]; ok {

		msg := fmt.Sprintf("Annotation %s deprecated in favor of %s, future versions won't support this annotation",
			gameserver.OctopsAnnotationIngressClassNameLegacy, gameserver.OctopsAnnotationIngressClassName)

		r.recorder.RecordEvent(gs, msg)
		runtime.Logger().Warn(strings.ToLower(msg))
	}
}

func (r *GameServerReconciler) MustReconcile(gs *agonesv1.GameServer) (bool, error) {
	if value, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationGameServerIngressReady); ok && len(value) > 0 {
		isReady, err := strconv.ParseBool(value)
		if err != nil {
			return true, err
		}

		return isReady == false, nil
	}

	return true, nil
}
