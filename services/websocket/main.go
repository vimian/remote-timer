package main

import (
	"github.com/vimian/remote-timer/services/websocket/config"
	"github.com/vimian/remote-timer/services/websocket/server"
)

func main() {
	config := config.LoadConfig()
	server.Listen(&config.Websocket)
}
