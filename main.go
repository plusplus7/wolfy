//go:build !remote

package main

import (
	"context"
	"fmt"
	"log"
	"wolfy/components"
	danmucomponent "wolfy/components/danmu"
	messagescomponent "wolfy/components/messages"
	servercomponent "wolfy/components/server"
	songscomponent "wolfy/components/songs"
	ticketscomponent "wolfy/components/tickets"
	"wolfy/server"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in main: %v\n", r)
		}
	}()

	fmt.Println("Starting program...")
	ctx := context.Background()
	manager, err := components.NewManager("./runtime/component.params.json")
	if err != nil {
		log.Printf("failed to initialize component manager: %v", err)
		return
	}

	serverComponent := servercomponent.NewServerComponent()
	songsComponent := songscomponent.NewSongsComponent()
	messagesComponent := messagescomponent.NewMessagesComponent()
	ticketsComponent := ticketscomponent.NewTicketsComponent(songsComponent, messagesComponent.MessageChan())
	danmuComponent := danmucomponent.NewDanmuComponent(ticketsComponent.TaskChan(), messagesComponent.MessageChan())

	for _, component := range []components.Component{
		serverComponent,
		songsComponent,
		messagesComponent,
		ticketsComponent,
		danmuComponent,
	} {
		if err := manager.Register(component); err != nil {
			log.Printf("failed to register component %s: %v", component.Name(), err)
			return
		}
	}

	_ = songsComponent.Start(ctx)
	_ = messagesComponent.Start(ctx)
	_ = ticketsComponent.Start(ctx)
	_ = danmuComponent.Start(ctx)
	_ = serverComponent.Start(ctx)

	s := server.NewLocalServer(manager, ticketsComponent, messagesComponent, danmuComponent)
	s.Spin()
}
