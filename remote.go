//go:build remote

package main

import (
	"log"
	"os"
	"wolfy/server"
)

func main() {
	akID := os.Getenv("BILIBILI_AK_ID")
	akSecret := os.Getenv("BILIBILI_AK_SECRET")
	if akID == "" || akSecret == "" {
		log.Println("cannot serve with empty BILIBILI_AK_ID or BILIBILI_AK_SECRET")
		return
	}
	r := server.NewRemoteSignatory(akID, akSecret)
	r.Register()
	r.Spin()
}
