package handlers

import "context"

type EventHandler interface {
	OnAdd(ctx context.Context, obj interface{}) error
	OnUpdate(ctx context.Context, oldObj interface{}, newObj interface{}) error
	OnDelete(ctx context.Context, obj interface{}) error
}
