package server

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
	"wolfy/model"
	"wolfy/service"
)

type LocalServer struct {
	TicketMaster    model.ITicketMaster
	MessageManager  *model.MessageManager
	MetadataManager *model.SystemInfoManager
	server          *http.Server

	taskChan chan *model.Task
	sysChan  chan model.SystemEvent
	game     string
}

func NewLocalServer() *LocalServer {
	return &LocalServer{}
}

func (l *LocalServer) Init(manager *model.SystemInfoManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("manager is nil")
	}

	localTicketsCheckPointPath := "./runtime/tickets.checkpoint.json"
	localMessagesCheckPointPath := "./runtime/messages.checkpoint.json"
	localSongDBPath := "./static/" + manager.Get().Game + "/songs.json"

	l.server = l.Register()
	l.TicketMaster = service.NewMaimaiTicketMaster(localSongDBPath, localTicketsCheckPointPath, 12)
	l.MessageManager = model.NewMessageManager(localMessagesCheckPointPath, 3, 10*time.Second)
	l.MetadataManager = manager
	l.game = manager.Get().Game

	return "ready to spin", nil
}

func (l *LocalServer) taskRoutine(ctx context.Context, tasker chan *model.Task) {
	for {
		select {
		case task := <-tasker:
			_, err := l.taskHandler(task)
			if err != nil {
				log.Printf("task handle err %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (l *LocalServer) taskHandler(task *model.Task) (msg string, err error) {
	if task == nil {
		return "", nil
	}
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
		case model.CommandClearTickets:
			msg, err = l.TicketMaster.ClearTickets(caller)
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
	FrontendEventPick           = "pick"
	FrontendEventClickCoverInfo = "click_cover_info"
	FrontendEventClickGenreInfo = "click_genre_info"
	FrontendEventClickSongInfo  = "click_song_info"
	FrontendEventClickCreator   = "click_creator"
	FrontendEventReboot         = "reboot"
	FrontendEventClearAllData   = "clear_all_data"
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
	case FrontendEventPick:
		command = model.CommandPick
	case FrontendEventClickCoverInfo:
		command = model.CommandFinish
	case FrontendEventClickGenreInfo:
		command = model.CommandNextRank
	case FrontendEventClickSongInfo:
		command = model.CommandNextLevel
	case FrontendEventClickCreator:
		command = model.CommandPick
	case FrontendEventClearAllData:
		command = model.CommandClearTickets
	case FrontendEventReboot:
		l.sysChan <- model.SystemReboot
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
			Image:     "//" + c.Request.Host + "/static/" + l.game + "/covers/" + ticket.GetCoverPath(),
			CoverInfo: ticket.GetCoverInfo(),
			GenreInfo: ticket.GetGenreInfo(),
			SongInfo:  ticket.GetSongInfo(),
		})
	})

	c.JSON(200, gin.H{"data": result})
}

func (l *LocalServer) Metadata(c *gin.Context) {
	c.JSON(200, gin.H{"data": l.MetadataManager.Get()})
}

func (l *LocalServer) SetMetadata(c *gin.Context) {
	var req model.SystemInfo
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}

	err := l.MetadataManager.Save(req)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	l.sysChan <- model.SystemReboot
	c.JSON(200, gin.H{"data": "ok"})
}

func (l *LocalServer) Register() *http.Server {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			fmt.Println(origin)
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	router.Static("/static", "./static")
	router.GET("/event/:caller/:event/:content", l.Event)
	router.GET("/messages", l.Message)
	router.GET("/tickets", l.Tickets)
	router.GET("/metadata", l.Metadata)
	router.POST("/metadata", l.SetMetadata)
	return &http.Server{
		Addr:    ":41377",
		Handler: router,
	}
}

func (l *LocalServer) Spin(ctx context.Context, taskChan chan *model.Task, sysChan chan model.SystemEvent) (string, error) {
	l.taskChan = taskChan
	l.sysChan = sysChan

	if l.taskChan != nil {
		go l.taskRoutine(ctx, taskChan)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("LocalServer app shutting down")
				err := l.server.Shutdown(ctx)
				if err != nil {
					log.Printf("LocalServer server shutdown error %v\n", err)
				}
				return
			}
		}
	}()
	go func() {
		err := l.server.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()
	return "spinning", nil
}
