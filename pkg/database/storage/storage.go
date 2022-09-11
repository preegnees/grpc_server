package storage

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
	"sort"

	m "streaming/pkg/models"
)

var ErrInvalidIdChannelWhenRemove error = fmt.Errorf("Ошибка при удалении пира, такого канала не существует")
var ErrInvalidPeerWhenRemove error = fmt.Errorf("Ошибка при удалении пира, такого пира не существует")
var ErrInvalidAllowedNamesWhenSave error = fmt.Errorf("Ошибка при сохранении пира, allowedNames подключающегося клиентя не соответсвует allowedNames клиента в базе")

// Проверка на соответсвии интерфейсу
var _ m.IStreamStorage = (*storage)(nil)

// storage. Структура хранилища
type storage struct {
	// Хранение Id каналов и их пиров, которые состоят их пиров и токенов
	streams map[m.IdChannel](map[m.Peer]m.Token)
	// Хранение токенов и каналов
	chans   map[m.Token](chan map[m.Peer]struct{})
	// Мьютекс для сохранения и удаления
	mx      sync.Mutex
}

// NewStorage. Функция получения хранилища
func NewStorage() m.IStreamStorage {
	
	strg := make(map[m.IdChannel](map[m.Peer]m.Token))
	chs := make(map[m.Token](chan map[m.Peer]struct{}))
	return &storage{
		streams: strg,
		chans: chs,
	}
}

// SavePeer. Сохранение пира при подключении
func (s *storage) SavePeer(peer m.Peer) (<-chan map[m.Peer]struct{}, error) {

	s.mx.Lock()
	defer s.mx.Unlock()

	// Получаем все пиры, связанные с каналом того пира, который хочет подключится. Если его нет, то создаем 
	peers, ok := s.streams[peer.IdChannel]
	if !ok {
		peers = make(map[m.Peer]m.Token)
	}

	// Проверяем на тот случай если в хранилище осталось предыдущее подключение, если оно есть, то удаляем
	for p, t := range peers {
		if p.Name == peer.Name {
			delete(s.chans, t)
			delete(peers, p)
		}
	}

	// Проверка на тот случай если, новый клиент будет иметь набор друзей, отличный от того, что есть уже в базе
	if ok {
		for p := range peers {
			allowedPeersInDbSplited := strings.Split(p.AllowedNames,",")
			sort.Strings(allowedPeersInDbSplited)
			allowedPeersInDb := strings.Join(allowedPeersInDbSplited, "")

			allowedPeersInThisSplited := strings.Split(peer.AllowedNames, ",")
			sort.Strings(allowedPeersInThisSplited)
			allowedPeersInThis := strings.Join(allowedPeersInThisSplited, "")
			
			if allowedPeersInDb != allowedPeersInThis {
				return nil, ErrInvalidAllowedNamesWhenSave
			}
			break
		}
	}

	// Создание канала
	ch := make((chan map[m.Peer]struct{}))
	
	// Создаение токена, который будет связывать хранилище с клиентами и их каналами
	token := m.Token(fmt.Sprintf("%d",rand.Int() + int(time.Now().UnixNano()) + rand.Int()))
	// Сохранение токена и канала в хронилище каналов
	s.chans[token] = ch
	// Сохранение пира и токена
	peers[peer] = token
	// Сохранение обратно всего в общее хранилище
	s.streams[peer.IdChannel] = peers

	// Рассылка всем, так как подключился новый клиент
	go s.sendPeers(peer.IdChannel)

	return ch, nil
}

// DeletePeer. Удаление пира при отключении
func (s *storage) DeletePeer(peer m.Peer) error {

	s.mx.Lock()
	defer s.mx.Unlock()

	// Получение пиров, который связаны с данном каналом
	peers, ok := s.streams[peer.IdChannel]
	if !ok {
		return ErrInvalidIdChannelWhenRemove
	}

	// Получение токена, который свзан с данным пиром
	token, ok := peers[peer]
	if !ok {
		return ErrInvalidPeerWhenRemove // нужны ли тут вообще ошибки???
	}

	// Пытаемся достать канал, который связан с токеном, закрываем его и удаляем
	ch, ok := s.chans[token]
	if ok {
		close(ch)
		ch = nil
		delete(s.chans, token)
	}
	
	// Удаление пира
	delete(peers, peer)

	// Обратное сохранение пиров под idch
	s.streams[peer.IdChannel] = peers

	// Рассылка всем об удалении
	go s.sendPeers(peer.IdChannel)

	return nil
}

// sendPeers. Вызывается каждый раз, когда происходят изменения в храненилищах
func (s *storage) sendPeers(idCh m.IdChannel) {

	// Пытаемся получить все пиры по idCh, если таких нет, то выходм (их может не быть???)
	peers, ok := s.streams[idCh]
	if !ok {
		return
	}

	// Тут происходит создание двух мап, для хранения раздельно пиров и каналов этих пиров
	ps := make(map[m.Peer]struct{})                    
	chs := make(map[chan map[m.Peer]struct{}]struct{})
	for p, t := range peers {
		ps[p] = struct{}{}
		ch, ok := s.chans[t]
		if ok {
			chs[ch] = struct{}{}
		}
	}

	// Тут осуществляется рассылка всем клиентам об изменениях
	go func() {
		for ch := range chs {
			go func(ch chan<- map[m.Peer]struct{}) {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println("канал почему то закрыт, err:=", err)
					}
				}()
				ch <- ps
			}(ch)
		}
	}()
}