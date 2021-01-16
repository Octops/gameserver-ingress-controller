package handlers

type EventHandler interface {
	OnAdd(obj interface{}) error
	OnUpdate(oldObj interface{}, newObj interface{}) error
	OnDelete(obj interface{}) error
}
