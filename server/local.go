package server

import (
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"strconv"
	"time"
	"wolfy/components"
	danmucomponent "wolfy/components/danmu"
	messagescomponent "wolfy/components/messages"
	ticketscomponent "wolfy/components/tickets"
	"wolfy/model"
)

type LocalServer struct {
	router   *gin.Engine
	manager  *components.Manager
	tickets  *ticketscomponent.TicketsComponent
	messages *messagescomponent.MessagesComponent
	danmu    *danmucomponent.DanmuComponent
}

func NewLocalServer(manager *components.Manager, tickets *ticketscomponent.TicketsComponent, messages *messagescomponent.MessagesComponent, danmu ...*danmucomponent.DanmuComponent) *LocalServer {
	l := &LocalServer{
		router:   gin.Default(),
		manager:  manager,
		tickets:  tickets,
		messages: messages,
	}
	if len(danmu) > 0 {
		l.danmu = danmu[0]
	}
	l.Register()

	return l
}

func (l *LocalServer) taskHandler(task *model.Task) (msg string, err error) {
	if l.tickets == nil {
		return "", fmt.Errorf("tickets component is not registered")
	}
	return l.tickets.HandleTask(task)
}

func (l *LocalServer) Event(c *gin.Context) {
	task := ticketscomponent.TaskFromFrontend(c.Param("caller"), c.Param("event"), c.Param("content"))
	msg, err := l.taskHandler(task)

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
	if l.messages == nil {
		c.JSON(503, gin.H{"msg": "messages component is not registered"})
		return
	}
	var result = &GetMessagesResponse{
		Messages: make([]model.Message, 0),
	}
	if err := l.messages.ForEachMessage(func(message *model.Message) {
		result.Messages = append(result.Messages, *message)
	}); err != nil {
		c.JSON(503, gin.H{"msg": err.Error()})
		return
	}
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
	if l.tickets == nil {
		c.JSON(503, gin.H{"msg": "tickets component is not registered"})
		return
	}
	var result GetTicketsResponse
	if err := l.tickets.ForEachTicket(func(ticket model.ITicket) {
		result.Tickets = append(result.Tickets, TicketItem{
			Title:     ticket.GetTitle(),
			Keyword:   ticket.GetKeyword(),
			Creator:   ticket.GetCreator(),
			Image:     ticket.GetCoverPath(),
			CoverInfo: ticket.GetCoverInfo(),
			GenreInfo: ticket.GetGenreInfo(),
			SongInfo:  ticket.GetSongInfo(),
		})
	}); err != nil {
		c.JSON(503, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": result})
}

type updateComponentParamsRequest struct {
	Params map[string]string `json:"params"`
}

type RemoteDanmuConfig struct {
	RemoteBaseURL string `json:"remote_base_url"`
	AppID         int64  `json:"app_id"`
	AnchorCode    string `json:"anchor_code"`
}

type UpdateRemoteDanmuConfigRequest struct {
	Config RemoteDanmuConfig `json:"config"`
}

type LocalDanmuStatus struct {
	Status  components.Status `json:"status"`
	Config  RemoteDanmuConfig `json:"config"`
	LastSeq int64             `json:"last_seq"`
	Error   string            `json:"error,omitempty"`
}

type ComponentEventTypesResponse struct {
	Types []components.ComponentEventTypeInfo `json:"types"`
}

func (l *LocalServer) SysInfo(c *gin.Context) {
	c.JSON(200, gin.H{"data": l.manager.Snapshots()})
}

func (l *LocalServer) ComponentEventTypes(c *gin.Context) {
	c.JSON(200, gin.H{"data": ComponentEventTypesResponse{Types: components.ComponentEventTypes()}})
}

func (l *LocalServer) DanmuStatus(c *gin.Context) {
	status, err := l.danmuStatus()
	if err != nil {
		c.JSON(503, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": status})
}

func (l *LocalServer) UpdateDanmuConfig(c *gin.Context) {
	if l.danmu == nil {
		c.JSON(503, gin.H{"msg": "danmu component is not registered"})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	var req UpdateRemoteDanmuConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	if req.Config == (RemoteDanmuConfig{}) {
		if err := json.Unmarshal(body, &req.Config); err != nil {
			c.JSON(400, gin.H{"msg": err.Error()})
			return
		}
	}
	_, err = l.manager.UpdateParams(danmucomponent.DanmuComponentName, map[string]string{
		danmucomponent.ParamRemoteBaseURL: req.Config.RemoteBaseURL,
		danmucomponent.ParamAppID:         strconv.FormatInt(req.Config.AppID, 10),
		danmucomponent.ParamAnchorCode:    req.Config.AnchorCode,
	})
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	status, err := l.danmuStatus()
	if err != nil {
		c.JSON(503, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": status})
}

func (l *LocalServer) StartDanmu(c *gin.Context) {
	if l.danmu == nil {
		c.JSON(503, gin.H{"msg": "danmu component is not registered"})
		return
	}
	if _, err := l.manager.Restart(c.Request.Context(), danmucomponent.DanmuComponentName); err != nil {
		status, _ := l.danmuStatus()
		c.JSON(400, gin.H{"msg": err.Error(), "data": status})
		return
	}
	status, err := l.danmuStatus()
	if err != nil {
		c.JSON(503, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": status})
}

func (l *LocalServer) StopDanmu(c *gin.Context) {
	if l.danmu == nil {
		c.JSON(503, gin.H{"msg": "danmu component is not registered"})
		return
	}
	if _, err := l.manager.Stop(c.Request.Context(), danmucomponent.DanmuComponentName); err != nil {
		status, _ := l.danmuStatus()
		c.JSON(400, gin.H{"msg": err.Error(), "data": status})
		return
	}
	status, err := l.danmuStatus()
	if err != nil {
		c.JSON(503, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": status})
}

func (l *LocalServer) danmuStatus() (LocalDanmuStatus, error) {
	if l.danmu == nil {
		return LocalDanmuStatus{}, fmt.Errorf("danmu component is not registered")
	}
	remoteBaseURL, appID, anchorCode := l.danmu.Config()
	snapshot := l.danmu.Snapshot()
	return LocalDanmuStatus{
		Status: snapshot.Status,
		Config: RemoteDanmuConfig{
			RemoteBaseURL: remoteBaseURL,
			AppID:         appID,
			AnchorCode:    anchorCode,
		},
		LastSeq: l.danmu.LastSeq(),
		Error:   snapshot.Error,
	}, nil
}

func (l *LocalServer) UpdateComponentParams(c *gin.Context) {
	name := c.Param("name")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	var req updateComponentParamsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	if req.Params == nil {
		req.Params = map[string]string{}
		if err := json.Unmarshal(body, &req.Params); err != nil {
			c.JSON(400, gin.H{"msg": err.Error()})
			return
		}
	}
	snapshot, err := l.manager.UpdateParams(name, req.Params)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": snapshot})
}

func (l *LocalServer) RestartComponent(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")
	snapshot, err := l.manager.Restart(ctx, name)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error(), "data": snapshot})
		return
	}
	c.JSON(200, gin.H{"data": snapshot})
}

func (l *LocalServer) StopComponent(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")
	if name == "server" {
		snapshot, _ := l.manager.Component(name)
		if snapshot == nil {
			c.JSON(400, gin.H{"msg": "unknown component: " + name, "data": components.ComponentSnapshot{}})
			return
		}
		c.JSON(400, gin.H{
			"msg":  "server stop is not supported from the HTTP API",
			"data": snapshot.Snapshot(),
		})
		return
	}
	snapshot, err := l.manager.Stop(ctx, name)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error(), "data": snapshot})
		return
	}
	c.JSON(200, gin.H{"data": snapshot})
}

func (l *LocalServer) Register() {
	l.router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:41377",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:41377",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	l.router.Static("/static", "./static")
	l.router.GET("/api/event/:caller/:event/:content", l.Event)
	l.router.GET("/api/messages", l.Message)
	l.router.GET("/api/tickets", l.Tickets)
	l.router.GET("/api/sysinfo", l.SysInfo)
	l.router.GET("/api/components", l.SysInfo)
	l.router.GET("/api/component-event-types", l.ComponentEventTypes)
	l.router.GET("/api/danmu", l.DanmuStatus)
	l.router.PATCH("/api/danmu", l.UpdateDanmuConfig)
	l.router.POST("/api/danmu/start", l.StartDanmu)
	l.router.POST("/api/danmu/stop", l.StopDanmu)
	l.router.PATCH("/api/components/:name/params", l.UpdateComponentParams)
	l.router.POST("/api/components/:name/restart", l.RestartComponent)
	l.router.POST("/api/components/:name/stop", l.StopComponent)
}

func (l *LocalServer) Spin() {
	err := OpenURL("http://localhost:41377/static/")
	if err != nil {
		log.Println(err)
	}
	err = l.router.Run("[::]:41377")

	if err != nil {
		log.Println(err)
	}
}
