package model

type ITicketMaster interface {
	AddTicket(creator string, keyword string) (string, error)
	FinishTicket(operator string, index int64) (string, error)
	ForEachTicket(fn func(ITicket))
	NextLevel(operator string, index int64) (string, error)
	NextRank(operator string, index int64) (string, error)
}

type ITicket interface {
	GetTitle() string
	GetKeyword() string
	GetCreator() string
	GetCoverPath() string

	GetCoverInfo() string
	GetGenreInfo() string
	GetSongInfo() string
}
