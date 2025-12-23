package server

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"wolfy/model"
	"wolfy/service"
)

type LocalServer struct {
	TicketMaster   model.ITicketMaster
	MessageManager *model.MessageManager
	router         *gin.Engine

	taskChan chan *model.Task
}

func NewLocalServer(songDB string, aliasPath string, taskChan chan *model.Task) *LocalServer {

	localTicketsCheckPointPath := "./runtime/tickets.checkpoint.json"
	localMessagesCheckPointPath := "./runtime/messages.checkpoint.json"

	l := &LocalServer{
		router:         gin.Default(),
		TicketMaster:   service.NewMaimaiTicketMaster(songDB, aliasPath, localTicketsCheckPointPath, 12),
		MessageManager: model.NewMessageManager(localMessagesCheckPointPath, 3, 10*time.Second),
		taskChan:       taskChan,
	}
	l.Register()
	if l.taskChan != nil {
		go l.taskRoutine(l.taskChan)
	}

	return l
}

func (l *LocalServer) taskRoutine(tasker chan *model.Task) {
	for {
		task := <-tasker
		if task == nil { // shutdown
			break
		}
		_, err := l.taskHandler(task)
		if err != nil {
			return
		}
	}
}
func (l *LocalServer) taskHandler(task *model.Task) (msg string, err error) {
	var cmd = task.Command
	var caller = task.Caller
	var content = task.Content
	var index = task.Index

	if cmd == model.CommandPick {
		msg, err = l.TicketMaster.AddTicket(caller, content)
	} else {
		switch cmd {
		case model.CommandFinish:
			msg, err = l.TicketMaster.FinishTicket(caller, index)
		case model.CommandNextRank:
			msg, err = l.TicketMaster.NextRank(caller, index)
		case model.CommandNextLevel:
			msg, err = l.TicketMaster.NextLevel(caller, index)
		}
	}
	if err != nil {
		l.MessageManager.Push("err " + caller + " " + err.Error())
	} else {
		l.MessageManager.Push("inf " + caller + " " + msg)
	}
	return msg, err
}

const (
	FrontendEventClickCoverInfo = "click_cover_info"
	FrontendEventClickGenreInfo = "click_genre_info"
	FrontendEventClickSongInfo  = "click_song_info"
	FrontendEventClickCreator   = "click_creator"
	FrontendEventPick           = "pick"
)

func (l *LocalServer) Event(c *gin.Context) {
	caller := c.Param("caller")
	event := c.Param("event")
	content := c.Param("content")
	index, err := strconv.ParseInt(content, 10, 64)
	if err != nil {
		index = -1
	}
	var command string
	switch event {
	case FrontendEventClickCoverInfo:
		command = model.CommandFinish
	case FrontendEventClickGenreInfo:
		command = model.CommandNextRank
	case FrontendEventClickSongInfo:
		command = model.CommandNextLevel
	case FrontendEventPick:
		command = model.CommandPick
	case FrontendEventClickCreator:
		command = model.CommandPick
	}

	msg, err := l.taskHandler(&model.Task{
		Command: command,
		Caller:  caller,
		Content: content,
		Index:   index,
	})

	if err == nil {
		c.JSON(200, gin.H{"data": msg})
	} else {
		c.JSON(400, gin.H{"msg": err.Error()})
	}
}

type GetMessagesResponse struct {
	Messages []model.Message `json:"messages"`
}

func (l *LocalServer) Message(c *gin.Context) {
	var result = &GetMessagesResponse{
		Messages: make([]model.Message, 0),
	}
	l.MessageManager.ForEachMessage(func(message *model.Message) {
		result.Messages = append(result.Messages, *message)
	})
	c.JSON(200, gin.H{"data": result})
}

type TicketItem struct {
	Title   string `json:"title"`
	Keyword string `json:"keyword"`
	Creator string `json:"creator"`
	Image   string `json:"image"`

	CoverInfo string `json:"cover_info"`
	GenreInfo string `json:"genre_info"`
	SongInfo  string `json:"song_info"`
}

type GetTicketsResponse struct {
	Tickets []TicketItem `json:"tickets"`
}

func (l *LocalServer) Tickets(c *gin.Context) {
	var result GetTicketsResponse
	l.TicketMaster.ForEachTicket(func(ticket model.ITicket) {
		result.Tickets = append(result.Tickets, TicketItem{
			Title:     ticket.GetTitle(),
			Keyword:   ticket.GetKeyword(),
			Creator:   ticket.GetCreator(),
			Image:     ticket.GetCoverPath(),
			CoverInfo: ticket.GetCoverInfo(),
			GenreInfo: ticket.GetGenreInfo(),
			SongInfo:  ticket.GetSongInfo(),
		})
	})

	c.JSON(200, gin.H{"data": result})
}

func (l *LocalServer) Register() {
	l.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			fmt.Println(origin)
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	l.router.Static("/static", "./static")
	l.router.GET("/api/event/:caller/:event/:content", l.Event)
	l.router.GET("/api/messages", l.Message)
	l.router.GET("/api/tickets", l.Tickets)
}

func (l *LocalServer) Spin() {
	err := l.router.Run("[::]:41377")

	if err != nil {
		panic(err)
	}
}
