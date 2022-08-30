package main

import (
	"log"
	c "streaming/pkg/client/cli"
	m "streaming/pkg/models"
)


func main() {
	client := c.New()
	err := client.Run(
		m.CnfClient{
			Addr: "localhost:55001",
			AuthToken: "hello world",
			RequestTimeout: 15,
			KeepaliveInterval: 600,
			Reconnect: true,
			ReconnectTimeout: 15,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}