package bilibili

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"wolfy/components"
	"wolfy/model"
)

const openPlatformAPITimeout = 10 * time.Second

var (
	openPlatformHTTPHost   = OpenPlatformHttpHost
	openPlatformHTTPClient = &http.Client{Timeout: openPlatformAPITimeout}
)

type AppService struct {
	AppId       int64
	AnchorCode  string
	signatory   ISignatory
	taskChan    chan *model.Task
	messageChan chan<- string
	recorder    EventRecorder
}

type EventRecorder func(eventType components.ComponentEventType, codeLocation, message string)

func NewAppService(appId int64, anchorCode string, signatory ISignatory, messageChan chan<- string) *AppService {
	return &AppService{
		AppId:       appId,
		AnchorCode:  anchorCode,
		signatory:   signatory,
		taskChan:    make(chan *model.Task),
		messageChan: messageChan,
	}
}

func (a *AppService) SetEventRecorder(recorder EventRecorder) {
	a.recorder = recorder
}

func (a *AppService) Spin(ctx context.Context) chan *model.Task {
	var resp *BaseResp
	var err error
	for {
		if ctx.Err() != nil {
			return nil
		}
		resp, err = a.startApp(ctx)
		if err != nil {
			log.Printf("app service start failed, %v, restarting", err)
			if !sleepWithContext(ctx, time.Second) {
				return nil
			}
			continue
		}
		if resp == nil {
			log.Printf("app service start failed, %v, restarting", err)
			if !sleepWithContext(ctx, time.Second) {
				return nil
			}
			continue
		} else {
			log.Printf("app service spin success %v", resp.Message)
		}
		break
	}
	startAppRespData := &StartAppRespData{}
	err = json.Unmarshal(resp.Data, &startAppRespData)
	if err != nil {
		log.Printf("start app response unmarshal failed: %v", err)
		return nil
	}

	if startAppRespData == nil {
		log.Println("start app get msg err")
		return nil
	}

	if len(startAppRespData.WebsocketInfo.WssLink) == 0 {
		log.Println("start app websocket info get msg err")
		return nil
	}

	gameId := startAppRespData.GameInfo.GameId
	go func() {
		ticker := time.NewTicker(time.Second * 20)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, heartErr := a.appHeart(ctx, gameId)
				if heartErr != nil {
					log.Printf("app heart failed, %v\n", heartErr)
				}
			}
		}
	}()

	// 开启长连
	err = StartWebsocket(ctx,
		startAppRespData.WebsocketInfo.WssLink[0],
		startAppRespData.WebsocketInfo.AuthBody,
		a.taskChan,
		a.messageChan,
		a.recorder)
	if err != nil {
		log.Println(err)
		return nil
	}

	go func() {
		<-ctx.Done()
		endCtx, cancel := context.WithTimeout(context.Background(), openPlatformAPITimeout)
		defer cancel()
		_, endErr := a.endApp(endCtx, gameId, a.AppId)
		if endErr != nil {
			log.Printf("end app failed: %v", endErr)
		}
	}()
	return a.taskChan
}

func sleepWithContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (a *AppService) startApp(ctx context.Context) (resp *BaseResp, err error) {
	startAppReq := StartAppRequest{
		Code:  a.AnchorCode,
		AppId: a.AppId,
	}
	reqJson, _ := json.Marshal(startAppReq)
	return a.apiRequest(ctx, string(reqJson), "/v2/app/start")
}

// AppHeart app心跳
func (a *AppService) appHeart(ctx context.Context, gameId string) (resp *BaseResp, err error) {
	appHeartbeatReq := AppHeartbeatReq{
		GameId: gameId,
	}
	reqJson, _ := json.Marshal(appHeartbeatReq)
	return a.apiRequest(ctx, string(reqJson), "/v2/app/heartbeat")
}

// EndApp 关闭app
func (a *AppService) endApp(ctx context.Context, gameId string, appId int64) (resp *BaseResp, err error) {
	endAppReq := EndAppRequest{
		GameId: gameId,
		AppId:  appId,
	}
	reqJson, _ := json.Marshal(endAppReq)
	return a.apiRequest(ctx, string(reqJson), "/v2/app/end")
}

// apiRequest http request demo方法
func (a *AppService) apiRequest(ctx context.Context, reqJson, requestUrl string) (*BaseResp, error) {
	header, err := a.signatory.Sign(reqJson)
	if err != nil {
		return nil, fmt.Errorf("sign err: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		fmt.Sprintf("%s%s", openPlatformHTTPHost, requestUrl),
		bytes.NewBuffer([]byte(reqJson)))
	if err != nil {
		return nil, err
	}
	req.Header = header.ToHTTPHeader()
	resp, err := openPlatformHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("bilibili api %s returned %s", requestUrl, resp.Status)
	}
	var result BaseResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
