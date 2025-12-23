package bilibili

import (
	"strconv"
	"strings"
	"wolfy/model"
)

const (
	KeyWordPick      = "点歌"
	KeyWordRePick    = "换歌"
	KeyWordNextLevel = "换谱"
	KeyWordDelete    = "删除"
)

func parseDanmu(caller, message string) *model.Task {
	var command string
	var index int64 = -1
	message = strings.TrimSpace(message)
	if strings.HasPrefix(message, KeyWordPick) {
		command = model.CommandPick
		message = strings.TrimLeft(message, KeyWordPick)
	} else {
		if strings.HasPrefix(message, KeyWordRePick) {
			command = model.CommandNextRank
			message = strings.TrimLeft(message, KeyWordPick)
		} else if strings.HasPrefix(message, KeyWordNextLevel) {
			command = model.CommandNextLevel
			message = strings.TrimLeft(message, KeyWordPick)
		} else if strings.HasPrefix(message, KeyWordDelete) {
			command = model.CommandFinish
			message = strings.TrimLeft(message, KeyWordDelete)
		} else {
			return nil
		}
		message = strings.TrimSpace(message)
		parseInt, err := strconv.ParseInt(message, 10, 64)
		if err == nil {
			index = parseInt - 1
			if index < 0 {
				index = 1
			}
		} else {
			return nil
		}

	}
	return &model.Task{
		Command: command,
		Caller:  caller,
		Content: message,
		Index:   index,
	}
}
