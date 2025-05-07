package bilibili

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"wolfy/model"
	"wolfy/service"
)

type AppService struct {
	AppID      int64
	GameID     string
	AnchorCode string
	signatory  ISignatory
	sys        *model.SystemInfoManager
}

func NewAppService() service.IService {
	return &AppService{}
}

func (a *AppService) Init(sys *model.SystemInfoManager) (string, error) {
	info := sys.Get()
	if info.AppID == 0 || info.AnchorCode == "" {
		return "", fmt.Errorf("app id, game or anchor code is empty")
	}
	a.AppID = info.AppID
	a.AnchorCode = info.AnchorCode
	a.GameID = info.BilibiliGameID
	a.sys = sys

	if info.BilibiliAccessKey != "" && info.BilibiliAccessSecret != "" {
		a.signatory = NewLocalSignatory(info.BilibiliAccessKey, info.BilibiliAccessSecret)
		return "running with local signatory", nil
	} else {
		a.signatory = NewRemoteSignatory(info.RemoteSignatoryAddr, a.AnchorCode)
		return "running with remote signatory", nil
	}
}

func (a *AppService) tryEndApp() {
	if a.GameID == "" || a.AppID == 0 {
		return
	}
	_, err := a.endApp(a.GameID, a.AppID)
	if err != nil {
		log.Println("error when shutting down", err)
	} else {
		info := a.sys.Get()
		info.BilibiliGameID = ""
		err = a.sys.Save(info)
		if err != nil {
			log.Println("error when saving metadata", err)
		}
	}
}

func (a *AppService) Spin(ctx context.Context, taskChan chan *model.Task, _ chan model.SystemEvent) (string, error) {
	if a.GameID != "" {
		log.Printf("try end app %s\n", a.GameID)
		a.tryEndApp()
	}

	resp, err := a.startApp()
	if err != nil {
		return "", err
	}
	startAppRespData := &StartAppRespData{}
	err = json.Unmarshal(resp.Data, &startAppRespData)
	if err != nil {
		return "", err
	}

	if startAppRespData == nil {
		return "", fmt.Errorf("failed to get app start message %v %v", *resp, err)
	}

	if len(startAppRespData.WebsocketInfo.WssLink) == 0 {
		return "", fmt.Errorf("failed to get websocket link")
	}

	a.GameID = startAppRespData.GameInfo.GameId
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("app shutting down")
				a.tryEndApp()
				log.Println("app shutting down done")
				return
			default:
				time.Sleep(time.Second * 20)
				_, _ = a.appHeart(a.GameID)
			}
		}
	}()

	log.Println("start app websocket")
	// 开启长连
	err = StartWebsocket(
		ctx,
		startAppRespData.WebsocketInfo.WssLink[0],
		startAppRespData.WebsocketInfo.AuthBody,
		taskChan)
	if err != nil {
		return "", fmt.Errorf("failed to start websocket")
	}

	return "spinning", nil
}

func (a *AppService) startApp() (resp *BaseResp, err error) {
	startAppReq := StartAppRequest{
		Code:  a.AnchorCode,
		AppId: a.AppID,
	}
	reqJson, _ := json.Marshal(startAppReq)
	return a.apiRequest(string(reqJson), "/v2/app/start")
}

// AppHeart app心跳
func (a *AppService) appHeart(gameId string) (resp *BaseResp, err error) {
	appHeartbeatReq := AppHeartbeatReq{
		GameId: gameId,
	}
	reqJson, _ := json.Marshal(appHeartbeatReq)
	return a.apiRequest(string(reqJson), "/v2/app/heartbeat")
}

// EndApp 关闭app
func (a *AppService) endApp(gameId string, appId int64) (resp *BaseResp, err error) {
	endAppReq := EndAppRequest{
		GameId: gameId,
		AppId:  appId,
	}
	reqJson, _ := json.Marshal(endAppReq)
	return a.apiRequest(string(reqJson), "/v2/app/end")
}

// apiRequest http request demo方法
func (a *AppService) apiRequest(reqJson, requestUrl string) (*BaseResp, error) {
	header, err := a.signatory.Sign(reqJson)
	if err != nil {
		return nil, fmt.Errorf("sign err: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s%s", OpenPlatformHttpHost, requestUrl),
		bytes.NewBuffer([]byte(reqJson)))
	req.Header = header.ToHTTPHeader()

	if err != nil {
		return nil, err
	}
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	var result BaseResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
