package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wolfy/model"
	"wolfy/server"
	"wolfy/service"
	"wolfy/service/bilibili"
)

func main() {
	sysChan := make(chan model.SystemEvent)
	taskChan := make(chan *model.Task)

	sysInfo := model.NewSystemInfoManager()

	for {
		ctx, cancel := context.WithCancel(context.Background())

		{
			var bs service.IService
			bs = bilibili.NewAppService()
			status, err := bs.Init(sysInfo)
			if err != nil {
				log.Printf("init bilibili err: %v\n", err)
				sysInfo.SetServiceInfo("bilibili", status, err)
			} else {
				status, err = bs.Spin(ctx, taskChan, sysChan)
				log.Printf("spin bilibili status %v err: %v\n", status, err)
				sysInfo.SetServiceInfo("bilibili", status, err)
			}
		}
		{
			s := server.NewLocalServer()
			status, err := s.Init(sysInfo)
			if err != nil {
				log.Printf("init local server err: %v\n", err)
				sysInfo.SetServiceInfo("http", status, err)
			} else {
				status, err = s.Spin(ctx, taskChan, sysChan)
				log.Printf("spin local server status %v err: %v\n", status, err)
				sysInfo.SetServiceInfo("http", status, err)
			}
		}

		// 退出
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			for {
				sig := <-c
				switch sig {
				case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
					log.Printf("exit with syscall %v", sig)
					sysChan <- model.SystemExit
					return
				case syscall.SIGHUP:
				default:
					break
				}
			}
		}()

		for {
			event := <-sysChan
			if event == model.SystemExit {
				cancel()
				time.Sleep(5 * time.Second)
				return
			} else if event == model.SystemReboot {
				cancel()
				time.Sleep(5 * time.Second)
				break
			} else {
				cancel()
				time.Sleep(5 * time.Second)
				panic("unhandled default case")
			}
		}
	}
}
