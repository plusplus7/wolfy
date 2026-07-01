package bilibili

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"wolfy/model"
)

const remoteDanmuTimeout = 35 * time.Second

var remoteDanmuHTTPClient = &http.Client{Timeout: remoteDanmuTimeout}

type RemoteSignRequest struct {
	ReqJson    string `json:"req_json"`
	AnchorCode string `json:"anchor_code"`
}

type RemoteSignResponse struct {
	Header CommonHeader `json:"signed"`
}

type StartGameRequest struct {
	AppID      int64  `json:"app_id"`
	AnchorCode string `json:"anchor_code"`
	Force      bool   `json:"force,omitempty"`
}

type StopGameRequest struct {
	Reason string `json:"reason,omitempty"`
}

type GameSession struct {
	AnchorCode      string     `json:"anchor_code"`
	AppID           int64      `json:"app_id"`
	GameID          string     `json:"game_id"`
	Status          string     `json:"status"`
	StartedAt       string     `json:"started_at"`
	LastHeartbeatAt string     `json:"last_heartbeat_at"`
	LastSeq         int64      `json:"last_seq"`
	Anchor          AnchorInfo `json:"anchor,omitempty"`
	Error           string     `json:"error,omitempty"`
}

type StartGameResponse struct {
	Data GameSession `json:"data"`
}

type StopGameResponse struct {
	Data GameSession `json:"data"`
}

type DanmuEvent struct {
	Seq        int64       `json:"seq"`
	AnchorCode string      `json:"anchor_code"`
	MsgID      string      `json:"msg_id"`
	RoomID     int64       `json:"room_id"`
	Caller     string      `json:"caller"`
	UID        int64       `json:"uid"`
	Uface      string      `json:"uface"`
	Message    string      `json:"message"`
	Timestamp  int64       `json:"timestamp"`
	ReceivedAt string      `json:"received_at"`
	RawCmd     string      `json:"raw_cmd,omitempty"`
	Task       *model.Task `json:"task,omitempty"`
}

type PullDanmuResponse struct {
	Events  []DanmuEvent `json:"events"`
	NextSeq int64        `json:"next_seq"`
	HasMore bool         `json:"has_more"`
}

type RemoteDanmuClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRemoteDanmuClient(baseURL string) *RemoteDanmuClient {
	return &RemoteDanmuClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: remoteDanmuHTTPClient,
	}
}

func (c *RemoteDanmuClient) StartGame(ctx context.Context, req StartGameRequest) (*StartGameResponse, error) {
	var resp StartGameResponse
	err := c.doJSON(ctx, http.MethodPost, "/openapi/games", nil, req, &resp)
	return &resp, err
}

func (c *RemoteDanmuClient) GetGame(ctx context.Context, anchorCode string) (*StartGameResponse, error) {
	var resp StartGameResponse
	err := c.doJSON(ctx, http.MethodGet, "/openapi/games/"+url.PathEscape(anchorCode), nil, nil, &resp)
	return &resp, err
}

func (c *RemoteDanmuClient) StopGame(ctx context.Context, anchorCode string, req StopGameRequest) (*StopGameResponse, error) {
	var resp StopGameResponse
	err := c.doJSON(ctx, http.MethodDelete, "/openapi/games/"+url.PathEscape(anchorCode), nil, req, &resp)
	return &resp, err
}

func (c *RemoteDanmuClient) PullDanmu(ctx context.Context, anchorCode string, afterSeq int64, limit int, waitMS int) (*PullDanmuResponse, error) {
	query := url.Values{}
	query.Set("after_seq", strconv.FormatInt(afterSeq, 10))
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if waitMS > 0 {
		query.Set("wait_ms", strconv.Itoa(waitMS))
	}
	var resp PullDanmuResponse
	err := c.doJSON(ctx, http.MethodGet, "/openapi/games/"+url.PathEscape(anchorCode)+"/danmu", query, nil, &resp)
	return &resp, err
}

func (c *RemoteDanmuClient) doJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	if c.baseURL == "" {
		return fmt.Errorf("remote_base_url is empty")
	}
	var reader io.Reader
	if body != nil {
		marshaled, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewBuffer(marshaled)
	}
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reader)
	if err != nil {
		return err
	}
	req.Header.Set(ContentTypeHeader, JsonType)
	req.Header.Set(AcceptHeader, JsonType)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(resp.Body)
		if len(respBody) > 0 {
			return fmt.Errorf("remote danmu returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
		}
		return fmt.Errorf("remote danmu returned %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
