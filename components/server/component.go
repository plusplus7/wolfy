package server

import (
	"context"
	"errors"
	"wolfy/components"
)

type ServerComponent struct {
	*components.BaseComponent
}

func NewServerComponent() *ServerComponent {
	return &ServerComponent{BaseComponent: components.NewBaseComponent("server", nil)}
}

func (s *ServerComponent) Start(ctx context.Context) error {
	s.SetStatus(components.StatusRunning, nil)
	return nil
}

func (s *ServerComponent) Stop(ctx context.Context) error {
	s.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (s *ServerComponent) Restart(ctx context.Context) error {
	err := errors.New("server restart is not supported from the HTTP API")
	s.SetStatus(components.StatusError, err)
	return err
}
