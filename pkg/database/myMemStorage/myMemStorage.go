package mymemstorage

import (
	"fmt"
	"log"
	m "streaming/pkg/models"
)

type myStorage struct {
	streams map[string]m.Stream
	// peers   map[m.Peer]struct{}
}

var _ m.IStreamStorage = (*myStorage)(nil)

func (s *myStorage) SaveStream(streamNew m.Stream) (string, error) {
	streamOld, ok := s.streams[streamNew.IdChannel]
	if ok {
		log.Println(streamNew)
		log.Println(streamOld)
		if len(streamNew.Peers) != len(streamOld.Peers) {
			return "", fmt.Errorf("Одинаковый id канал, но разные значения, зачит вы где то ошиблись (разная длинна)")
		}
		count := 0
		l := len(streamNew.Peers)
		for _, pOld := range streamOld.Peers {
			for _, pNew := range streamNew.Peers {
				if pNew.Name == pOld.Name {
					count++
				}
			}
		}
		if l == count {
			return "Такой стрим уже есть", nil
		}
		return "", nil
	} else {
		s.streams[streamNew.IdChannel] = streamNew
		log.Println("Было сохранено:", streamNew)
		return "", nil
	}
}

func (s *myStorage) SavePeer(idChannel string, peer m.Peer) error {
	return nil
}

func (s *myStorage) SetFiledConnected(idChannel string, peer m.Peer, is bool) error {
	return nil
}

func (s *myStorage) GetStreams() (strm chan<- m.Stream) {
	return nil
}

func New() m.IStreamStorage {
	return &myStorage{
		streams: make(map[string]m.Stream, 0),
		// peers:   make(map[m.Peer]struct{}, 0),
	}
}
