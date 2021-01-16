package handlers

import (
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/sirupsen/logrus"
)

type EventHandler interface {
	OnAdd(obj interface{}) error
	OnUpdate(oldObj interface{}, newObj interface{}) error
	OnDelete(obj interface{}) error
}

type GameSeverEventHandler struct {
	*logrus.Entry
}

func NewGameSeverEventHandler() *GameSeverEventHandler {
	return &GameSeverEventHandler{
		runtime.Logger(),
	}
}

func (g *GameSeverEventHandler) OnAdd(obj interface{}) error {
	g.Info(obj)

	return nil
}

func (g *GameSeverEventHandler) OnUpdate(oldObj interface{}, newObj interface{}) error {
	g.Info(newObj)

	return nil
}

func (g *GameSeverEventHandler) OnDelete(obj interface{}) error {
	g.Info(obj)

	return nil
}
