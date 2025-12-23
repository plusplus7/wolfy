package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"wolfy/server"
	"wolfy/service/bilibili"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in main: %v\n", r)
		}
	}()

	fmt.Println("Starting program...")
	akID := os.Getenv("BILIBILI_AK_ID")
	akSecret := os.Getenv("BILIBILI_AK_SECRET")

	anchorCode := os.Getenv("ANCHOR_CODE")
	appIDStr := os.Getenv("APP_ID")
	songPackage := os.Getenv("SONG_PACKAGE_PATH")
	aliasFile := os.Getenv("ALIAS_FILE_PATH")

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
	if bilibiliChan == nil {
		panic(fmt.Errorf("bilibiliApp.Spin() returned nil"))
	}
	s := server.NewLocalServer(songPackage, aliasFile, bilibiliChan)
	s.Spin()
}
