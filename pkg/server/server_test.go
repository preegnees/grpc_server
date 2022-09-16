package server

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	model "streaming/pkg/models"

	client "streaming/pkg/client"
	// "go.uber.org/goleak"
)

var server = New()

func TestMain(m *testing.M) {
	fmt.Println("Инициализация")
	defer server.Stop()
	go func() {
		err := server.Run(
			model.CnfServer{
				Addr:            "localhost:55001",
				ShutdownTimeout: 2,
				CertPem:         "C:\\Users\\secrr\\Desktop\\my_streaming\\tls\\cert.pem",
				KeyPem:          "C:\\Users\\secrr\\Desktop\\my_streaming\\tls\\key.pem",
			},
		)
		if err != nil {
			log.Fatal(err)
		}
	}()
	fmt.Println("Сервер инициализирван")
	exitEval := m.Run()
	fmt.Println("Остановка")
	os.Exit(exitEval)
}

func TestXxx(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cli1 := client.New()
	cli2 := client.New()
	go func() {
		err := cli1.Run(
			model.CnfClient{
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
	}()


	go func() {
		err := cli2.Run(
			model.CnfClient{
				Addr:              "localhost:55001",
				RequestTimeout:    15,
				KeepaliveInterval: 600,
				Reconnect:         true,
				ReconnectTimeout:  15,
				IdChannel:         "1234567890",
				Name:              "Name1",
				AllowedNames:      "Name1, Name2, Name4",
			},
		)
		if err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(6 * time.Second)
	cli1.Stop()
	cli2.Stop()
}
