package service

import (
	"context"
	"wolfy/model"
)

type IService interface {
	Init(metadata *model.SystemInfoManager) (string, error)
	Spin(ctx context.Context, taskChan chan *model.Task, sysChan chan model.SystemEvent) (string, error)
}
