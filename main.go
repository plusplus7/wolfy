package main

import (
	"os"
	"strconv"
	"wolfy/server"
	"wolfy/service/bilibili"
)

func main() {
	akID := os.Getenv("BILIBILI_AK_ID")
	akSecret := os.Getenv("BILIBILI_AK_SECRET")
	anchorCode := os.Getenv("ANCHOR_CODE")
	appIDStr := os.Getenv("APP_ID")
	game := os.Getenv("GAME")

	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		panic(err)
	}
	var signatory bilibili.ISignatory
	if akID != "" && akSecret != "" {
		signatory = bilibili.NewLocalSignatory(akID, akSecret)
	} else {
		signatory = bilibili.NewRemoteSignatory("https://plusplus7.com:42376", akSecret)
	}
	bilibiliApp := bilibili.NewAppService(
		int64(appID),
		anchorCode,
		signatory,
	)
	bilibiliChan := bilibiliApp.Spin()
	s := server.NewLocalServer(game, bilibiliChan)
	s.Spin()
}
