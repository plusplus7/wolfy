package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
	"wolfy/components/danmu/bilibili"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	remoteDanmuDefaultLimit   = 100
	remoteDanmuMaxLimit       = 500
	remoteDanmuDefaultWaitMS  = 30000
	remoteDanmuMaxWaitMS      = 30000
	remoteDanmuBufferCapacity = 1024
)

type RemoteAppStarter func(ctx context.Context, appID int64, anchorCode string, eventSink chan<- *bilibili.DanmuEvent) (*bilibili.StartAppRespData, error)

type RemoteDanmuServer struct {
	router  *gin.Engine
	manager *RemoteDanmuManager
}

func NewRemoteDanmuServer(accessKeyID, accessKeySecret string) *RemoteDanmuServer {
	return NewRemoteDanmuServerWithStarter(func(ctx context.Context, appID int64, anchorCode string, eventSink chan<- *bilibili.DanmuEvent) (*bilibili.StartAppRespData, error) {
		app := bilibili.NewAppService(appID, anchorCode, bilibili.NewLocalSignatory(accessKeyID, accessKeySecret), nil)
		return app.SpinEvents(ctx, eventSink)
	})
}

func NewRemoteDanmuServerWithStarter(starter RemoteAppStarter) *RemoteDanmuServer {
	return &RemoteDanmuServer{
		manager: NewRemoteDanmuManager(starter, remoteDanmuBufferCapacity),
		router:  gin.Default(),
	}
}

func (r *RemoteDanmuServer) Spin() {
	err := r.router.Run("[::]:41376")
	if err != nil {
		log.Println(err)
	}
}

func (r *RemoteDanmuServer) Register() {
	r.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:41377", "http://127.0.0.1:3000", "http://127.0.0.1:3001", "http://127.0.0.1:41377"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.router.GET("/healthz", r.Health)
	r.router.POST("/openapi/games", r.StartGame)
	r.router.GET("/openapi/games/:anchor_code", r.GetGame)
	r.router.DELETE("/openapi/games/:anchor_code", r.StopGame)
	r.router.GET("/openapi/games/:anchor_code/danmu", r.PullDanmu)
}

func (r *RemoteDanmuServer) Health(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func (r *RemoteDanmuServer) StartGame(c *gin.Context) {
	var req bilibili.StartGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	session, err := r.manager.Start(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error(), "data": session})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (r *RemoteDanmuServer) GetGame(c *gin.Context) {
	session, err := r.manager.Get(c.Param("anchor_code"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (r *RemoteDanmuServer) StopGame(c *gin.Context) {
	var req bilibili.StopGameRequest
	_ = c.ShouldBindJSON(&req)
	session, err := r.manager.Stop(c.Param("anchor_code"), req.Reason)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (r *RemoteDanmuServer) PullDanmu(c *gin.Context) {
	afterSeq := parseInt64Query(c, "after_seq", 0)
	limit := parseIntQuery(c, "limit", remoteDanmuDefaultLimit)
	if limit <= 0 {
		limit = remoteDanmuDefaultLimit
	}
	if limit > remoteDanmuMaxLimit {
		limit = remoteDanmuMaxLimit
	}
	waitMS := parseIntQuery(c, "wait_ms", remoteDanmuDefaultWaitMS)
	if waitMS < 0 {
		waitMS = 0
	}
	if waitMS > remoteDanmuMaxWaitMS {
		waitMS = remoteDanmuMaxWaitMS
	}
	resp, err := r.manager.Poll(c.Request.Context(), c.Param("anchor_code"), afterSeq, limit, time.Duration(waitMS)*time.Millisecond)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64Query(c *gin.Context, key string, fallback int64) int64 {
	value := c.Query(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

type RemoteDanmuManager struct {
	mu       sync.Mutex
	starter  RemoteAppStarter
	sessions map[string]*remoteDanmuSession
	capacity int
}

func NewRemoteDanmuManager(starter RemoteAppStarter, capacity int) *RemoteDanmuManager {
	if capacity <= 0 {
		capacity = remoteDanmuBufferCapacity
	}
	return &RemoteDanmuManager{
		starter:  starter,
		sessions: map[string]*remoteDanmuSession{},
		capacity: capacity,
	}
}

func (m *RemoteDanmuManager) Start(ctx context.Context, req bilibili.StartGameRequest) (bilibili.GameSession, error) {
	if req.AppID == 0 {
		return bilibili.GameSession{}, errors.New("app_id is required")
	}
	if req.AnchorCode == "" {
		return bilibili.GameSession{}, errors.New("anchor_code is required")
	}

	m.mu.Lock()
	if current := m.sessions[req.AnchorCode]; current != nil && current.isRunning() && !req.Force {
		snapshot := current.snapshot()
		m.mu.Unlock()
		return snapshot, nil
	}
	if current := m.sessions[req.AnchorCode]; current != nil {
		current.stop("replaced")
	}
	sessionCtx, cancel := context.WithCancel(context.Background())
	session := newRemoteDanmuSession(req.AnchorCode, req.AppID, cancel, m.capacity)
	m.sessions[req.AnchorCode] = session
	m.mu.Unlock()

	eventSink := make(chan *bilibili.DanmuEvent, remoteDanmuDefaultLimit)
	startResp, err := m.starter(sessionCtx, req.AppID, req.AnchorCode, eventSink)
	if err != nil {
		cancel()
		session.setError(err)
		return session.snapshot(), err
	}
	session.markRunning(startResp)
	go session.consume(sessionCtx, eventSink)

	select {
	case <-ctx.Done():
		return session.snapshot(), ctx.Err()
	default:
		return session.snapshot(), nil
	}
}

func (m *RemoteDanmuManager) Get(anchorCode string) (bilibili.GameSession, error) {
	m.mu.Lock()
	session := m.sessions[anchorCode]
	m.mu.Unlock()
	if session == nil {
		return bilibili.GameSession{}, errors.New("unknown anchor_code: " + anchorCode)
	}
	return session.snapshot(), nil
}

func (m *RemoteDanmuManager) Stop(anchorCode string, reason string) (bilibili.GameSession, error) {
	m.mu.Lock()
	session := m.sessions[anchorCode]
	m.mu.Unlock()
	if session == nil {
		return bilibili.GameSession{}, errors.New("unknown anchor_code: " + anchorCode)
	}
	session.stop(reason)
	return session.snapshot(), nil
}

func (m *RemoteDanmuManager) Poll(ctx context.Context, anchorCode string, afterSeq int64, limit int, wait time.Duration) (bilibili.PullDanmuResponse, error) {
	m.mu.Lock()
	session := m.sessions[anchorCode]
	m.mu.Unlock()
	if session == nil {
		return bilibili.PullDanmuResponse{}, errors.New("unknown anchor_code: " + anchorCode)
	}
	return session.poll(ctx, afterSeq, limit, wait), nil
}

type remoteDanmuSession struct {
	mu              sync.Mutex
	anchorCode      string
	appID           int64
	gameID          string
	status          string
	startedAt       string
	lastHeartbeatAt string
	lastSeq         int64
	anchor          bilibili.AnchorInfo
	errText         string
	events          []bilibili.DanmuEvent
	capacity        int
	cancel          context.CancelFunc
	notify          chan struct{}
}

func newRemoteDanmuSession(anchorCode string, appID int64, cancel context.CancelFunc, capacity int) *remoteDanmuSession {
	now := time.Now().Format(time.RFC3339Nano)
	return &remoteDanmuSession{
		anchorCode:      anchorCode,
		appID:           appID,
		status:          "starting",
		startedAt:       now,
		lastHeartbeatAt: now,
		capacity:        capacity,
		cancel:          cancel,
		notify:          make(chan struct{}),
	}
}

func (s *remoteDanmuSession) markRunning(startResp *bilibili.StartAppRespData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if startResp != nil {
		s.gameID = startResp.GameInfo.GameId
		s.anchor = startResp.AnchorInfo
	}
	s.status = "running"
	s.errText = ""
	s.lastHeartbeatAt = time.Now().Format(time.RFC3339Nano)
	s.signalLocked()
}

func (s *remoteDanmuSession) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = "error"
	if err != nil {
		s.errText = err.Error()
	}
	s.signalLocked()
}

func (s *remoteDanmuSession) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status == "running"
}

func (s *remoteDanmuSession) stop(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.status = "stopped"
	if reason != "" {
		s.errText = reason
	} else {
		s.errText = ""
	}
	s.signalLocked()
}

func (s *remoteDanmuSession) consume(ctx context.Context, eventSink <-chan *bilibili.DanmuEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventSink:
			if event == nil {
				continue
			}
			s.appendEvent(*event)
		}
	}
}

func (s *remoteDanmuSession) appendEvent(event bilibili.DanmuEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSeq++
	event.Seq = s.lastSeq
	event.AnchorCode = s.anchorCode
	if event.ReceivedAt == "" {
		event.ReceivedAt = time.Now().Format(time.RFC3339Nano)
	}
	s.events = append(s.events, event)
	if len(s.events) > s.capacity {
		s.events = append([]bilibili.DanmuEvent(nil), s.events[len(s.events)-s.capacity:]...)
	}
	s.signalLocked()
}

func (s *remoteDanmuSession) poll(ctx context.Context, afterSeq int64, limit int, wait time.Duration) bilibili.PullDanmuResponse {
	deadline := time.Now().Add(wait)
	for {
		events, nextSeq, hasMore, notify := s.eventsAfter(afterSeq, limit)
		if len(events) > 0 || wait <= 0 || time.Now().After(deadline) {
			return bilibili.PullDanmuResponse{Events: events, NextSeq: nextSeq, HasMore: hasMore}
		}
		timer := time.NewTimer(time.Until(deadline))
		select {
		case <-ctx.Done():
			timer.Stop()
			return bilibili.PullDanmuResponse{Events: events, NextSeq: nextSeq, HasMore: hasMore}
		case <-notify:
			timer.Stop()
		case <-timer.C:
			return bilibili.PullDanmuResponse{Events: events, NextSeq: nextSeq, HasMore: hasMore}
		}
	}
}

func (s *remoteDanmuSession) eventsAfter(afterSeq int64, limit int) ([]bilibili.DanmuEvent, int64, bool, <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = remoteDanmuDefaultLimit
	}
	start := len(s.events)
	for i, event := range s.events {
		if event.Seq > afterSeq {
			start = i
			break
		}
	}
	if start >= len(s.events) {
		return nil, s.lastSeq, false, s.notify
	}
	end := start + limit
	hasMore := false
	if end < len(s.events) {
		hasMore = true
	} else {
		end = len(s.events)
	}
	events := append([]bilibili.DanmuEvent(nil), s.events[start:end]...)
	nextSeq := s.lastSeq
	if len(events) > 0 {
		nextSeq = events[len(events)-1].Seq
	}
	return events, nextSeq, hasMore, s.notify
}

func (s *remoteDanmuSession) snapshot() bilibili.GameSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return bilibili.GameSession{
		AnchorCode:      s.anchorCode,
		AppID:           s.appID,
		GameID:          s.gameID,
		Status:          s.status,
		StartedAt:       s.startedAt,
		LastHeartbeatAt: s.lastHeartbeatAt,
		LastSeq:         s.lastSeq,
		Anchor:          s.anchor,
		Error:           s.errText,
	}
}

func (s *remoteDanmuSession) signalLocked() {
	close(s.notify)
	s.notify = make(chan struct{})
}
