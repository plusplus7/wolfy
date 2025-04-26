package model

const (
	CommandPick      = "pick"
	CommandFinish    = "finish"
	CommandNextLevel = "next_level"
	CommandNextRank  = "next_rank"
)

type Task struct {
	Command string `json:"command"`
	Caller  string `json:"caller"`
	Content string `json:"content"`
	Index   int64  `json:"index"`
}
