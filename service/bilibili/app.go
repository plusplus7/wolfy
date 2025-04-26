package bilibili

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wolfy/model"
)

type AppService struct {
	AppId      int64
	AnchorCode string
	signatory  ISignatory
	taskChan   chan *model.Task
}

func NewAppService(appId int64, anchorCode string, signatory ISignatory) *AppService {
	return &AppService{
		AppId:      appId,
		AnchorCode: anchorCode,
		signatory:  signatory,
		taskChan:   make(chan *model.Task),
	}
}
func (a *AppService) Spin() chan *model.Task {
	resp, err := a.startApp()
	if err != nil {
		panic(err)
	}
	startAppRespData := &StartAppRespData{}
	err = json.Unmarshal(resp.Data, &startAppRespData)
	if err != nil {
		panic(err)
	}

	if startAppRespData == nil {
		log.Println("start app get msg err")
		return nil
	}

	if len(startAppRespData.WebsocketInfo.WssLink) == 0 {
		return nil
	}

	go func(gameId string) {
		for {
			time.Sleep(time.Second * 20)
			_, _ = a.appHeart(gameId)
		}
	}(startAppRespData.GameInfo.GameId)

	// 开启长连
	err = StartWebsocket(
		startAppRespData.WebsocketInfo.WssLink[0],
		startAppRespData.WebsocketInfo.AuthBody,
		a.taskChan)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// 退出
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for {
			s := <-c
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				log.Println("WebsocketClient exit")
				break
			case syscall.SIGHUP:
			default:
				break
			}
			//关闭应用
			_, err = a.endApp(startAppRespData.GameInfo.GameId, a.AppId)
			if err != nil {
				panic(err)
			}
		}
	}()
	return a.taskChan
}

func (a *AppService) startApp() (resp *BaseResp, err error) {
	startAppReq := StartAppRequest{
		Code:  a.AnchorCode,
		AppId: a.AppId,
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
