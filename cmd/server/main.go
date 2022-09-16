package main

import (
	"log"

	m "streaming/pkg/models"
	s "streaming/pkg/server/serv"
)

func main() {
	server := s.New()
	err := server.Run(
		m.CnfServer{
			Addr: "localhost:55001",
			ShutdownTimeout: 2,
			CertPem: "C:\\Users\\secrr\\Desktop\\my_streaming\\tls\\cert.pem",
			KeyPem: "C:\\Users\\secrr\\Desktop\\my_streaming\\tls\\key.pem",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}