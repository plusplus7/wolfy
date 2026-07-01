package bilibili

import (
	"strconv"
	"strings"
	"wolfy/components/danmaku"
	"wolfy/model"
)

func parseDanmu(caller, message string) *model.Task {
	return danmaku.ParseTask(caller, message)
}

func ParseRemoteDanmuTask(caller, message string) *model.Task {
	message = strings.TrimSpace(message)
	if strings.HasPrefix(message, danmaku.KeyWordPick) {
		content := strings.TrimSpace(strings.TrimPrefix(message, danmaku.KeyWordPick))
		return &model.Task{
			Command: model.CommandPick,
			Caller:  caller,
			Content: content,
			Index:   -1,
		}
	}

	var command string
	switch {
	case strings.HasPrefix(message, danmaku.KeyWordRePick):
		command = model.CommandNextRank
		message = strings.TrimPrefix(message, danmaku.KeyWordRePick)
	case strings.HasPrefix(message, danmaku.KeyWordNextLevel):
		command = model.CommandNextLevel
		message = strings.TrimPrefix(message, danmaku.KeyWordNextLevel)
	case strings.HasPrefix(message, danmaku.KeyWordDelete):
		command = model.CommandFinish
		message = strings.TrimPrefix(message, danmaku.KeyWordDelete)
	default:
		return nil
	}

	index, err := strconv.ParseInt(strings.TrimSpace(message), 10, 64)
	if err != nil {
		return nil
	}
	index--
	if index < 0 {
		index = 0
	}
	return &model.Task{
		Command: command,
		Caller:  caller,
		Content: "",
		Index:   index,
	}
}
