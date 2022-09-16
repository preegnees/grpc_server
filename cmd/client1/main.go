package main

import (
	"log"
	c "streaming/pkg/client"
	m "streaming/pkg/models"
)

func main() {
	client := c.New()
	err := client.Run(
		m.CnfClient{
			Addr:              "localhost:55001",
			RequestTimeout:    15,
			KeepaliveInterval: 600,
			Reconnect:         true,
			ReconnectTimeout:  15,
			IdChannel:         "1234567890",
			Name:              "Name1",
			AllowedNames:      "Name1, Name2, Name3",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
