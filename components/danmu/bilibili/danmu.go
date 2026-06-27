package bilibili

import (
	"wolfy/components/danmaku"
	"wolfy/model"
)

func parseDanmu(caller, message string) *model.Task {
	return danmaku.ParseTask(caller, message)
}
