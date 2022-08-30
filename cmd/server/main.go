package main

import (
	"log"

	m "streaming/pkg/models"
	s "streaming/pkg/server/serv"
)

const AUTH_TOKEN string = "hello world" 

func main() {
	server := s.New()
	err := server.Run(
		m.CnfServer{
			Addr: "localhost:55001",
			AuthToken: "hello world",
			Restart: false,
			ShutdownTimeout: 15,
			CertPem: "C:\\Users\\secrr\\Desktop\\my_streaming\\tls\\cert.pem",
			KeyPem: "C:\\Users\\secrr\\Desktop\\my_streaming\\tls\\key.pem",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}