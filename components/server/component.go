package server

import (
	"context"
	"errors"
	"wolfy/components"
)

const (
	ParamDanmuSource = "danmu_source"

	DanmuSourceDanmu   = "danmu"
	DanmuSourceBlivedm = "blivedm"
)

type ServerComponent struct {
	*components.BaseComponent
}

func NewServerComponent() *ServerComponent {
	return &ServerComponent{BaseComponent: components.NewBaseComponent("server", []string{
		ParamDanmuSource,
	})}
}

func (s *ServerComponent) DanmuSource() string {
	source := s.Params()[ParamDanmuSource]
	if source == "" {
		return DanmuSourceDanmu
	}
	return source
}

func (s *ServerComponent) ValidateParams(params map[string]string) error {
	if err := s.BaseComponent.ValidateParams(params); err != nil {
		return err
	}
	source, ok := params[ParamDanmuSource]
	if !ok || source == "" {
		return nil
	}
	if source != DanmuSourceDanmu && source != DanmuSourceBlivedm {
		return errors.New("danmu_source must be empty, danmu, or blivedm")
	}
	return nil
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
