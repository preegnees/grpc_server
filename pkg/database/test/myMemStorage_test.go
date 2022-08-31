package test

import (
	"log"
	"testing"

	mystor "streaming/pkg/database/myMemStorage"
	m "streaming/pkg/models"
)

func TestMyStorage(t *testing.T) {
	newdb := mystor.New()

	peers1 := []m.Peer{
		{
			Name: "radmir",
			Connected: false,
			Stream: nil,
		},
		{
			Name: "bogdan",
			Connected: false,
			Stream: nil,
		},
	}

	stream1 := m.Stream{
		IdChannel: "123",
		Peers: peers1,
	}

	peers2 := []m.Peer{
		{
			Name: "bogdan",
			Connected: false,
			Stream: nil,
		},
	}

	stream2 := m.Stream{
		IdChannel: "321",
		Peers: peers2,
	}

	stream3 := stream2

	msg, err := newdb.SaveStream(stream1)
	log.Println(msg)
	if err != nil {
		log.Println(err)
	}
	msg, err = newdb.SaveStream(stream2)
	log.Println(msg)
	if err != nil {
		log.Println(err)
	}
	msg, err = newdb.SaveStream(stream3)
	log.Println(msg)
	if err != nil {
		log.Println(err)
	}
}