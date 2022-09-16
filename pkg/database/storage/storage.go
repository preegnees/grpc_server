package storage

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	logrus "github.com/sirupsen/logrus"

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
	chans map[m.Token](chan map[m.Peer]struct{})
	// Мьютекс для сохранения и удаления
	mx sync.Mutex
	// Логгер
	log *logrus.Logger
}

func toString(peer m.Peer) string {
	return fmt.Sprintf(
		"Name=%s, IdChannel=%s, AllowedNames=%s, GrpcStream=%v", 
		peer.Name, peer.IdChannel, peer.AllowedNames, peer.GrpcStream,
	)
}

// NewStorage. Получение хранилища
func NewStorage(logger *logrus.Logger) m.IStreamStorage {

	// if Environment == "production" {
	// 	log.SetFormatter(&log.JSONFormatter{})
	// } else {
	// 	// The TextFormatter is default, you don't actually have to do this.
	// 	log.SetFormatter(&log.TextFormatter{})
	// }
	// log.SetLevel(log.DebugLevel)
	// log.New().Out = os.Stdout
	// log.SetFormatter(&log.TextFormatter{})
	// log.Debug("debug")

	strg := make(map[m.IdChannel](map[m.Peer]m.Token))
	chs := make(map[m.Token](chan map[m.Peer]struct{}))
	return &storage{
		streams: strg,
		chans:   chs,
		log: logger,
	}
}

// SavePeer. Сохранение пира и рассылка всем клиентам обновленной мапы с пирами
func (s *storage) SavePeer(peer m.Peer) (<-chan map[m.Peer]struct{}, error) {

	s.mx.Lock()
	defer s.mx.Unlock()
	s.log.Debugf("SavePeer. Вызвана функция сохраненния, пир(%s)\n", toString(peer))

	// Получаем все пиры, связанные с каналом того пира, который хочет подключится. Если его нет, то создаем
	peers, ok := s.streams[peer.IdChannel]
	if !ok {
		peers = make(map[m.Peer]m.Token)
		s.log.Debugf("SavePeer. Пир (%s) не был найден в стримах\n", toString(peer))
	}

	// Проверка на тот случай если, новый клиент будет иметь набор друзей, отличный от того, что есть уже в базе
	if ok {
		for p := range peers {
			allowedPeersInDbSplited := strings.Split(strings.ReplaceAll(p.AllowedNames, " ", ""), ",")
			sort.Strings(allowedPeersInDbSplited)
			allowedPeersInDb := strings.Join(allowedPeersInDbSplited, "")

			allowedPeersInThisSplited := strings.Split(strings.ReplaceAll(peer.AllowedNames, " ", ""), ",")
			sort.Strings(allowedPeersInThisSplited)
			allowedPeersInThis := strings.Join(allowedPeersInThisSplited, "")

			if allowedPeersInDb != allowedPeersInThis {
				s.log.Warnf("SavePeer. AllowedPeers не совпало у пира (%s)", toString(peer))
				return nil, ErrInvalidAllowedNamesWhenSave
			}
			break
		}
	}

	// Проверяем на тот случай если в хранилище осталось предыдущее подключение, если оно есть, то удаляем
	for p, t := range peers {
		if p.Name == peer.Name {
			s.log.Warnf("SavePeer. В хранилище осталось подключение с имененм:%s\n", p.Name)
			delete(s.chans, t)
			delete(peers, p)
			go s.sendPeers(peer.IdChannel)
		}
	}

	// Создание канала
	ch := make((chan map[m.Peer]struct{}))

	// Создаение токена, который будет связывать хранилище с клиентами и их каналами
	token := m.Token(fmt.Sprintf("%d", rand.Int()+int(time.Now().UnixNano())+rand.Int()))
	// Сохранение токена и канала в хранилище каналов
	s.chans[token] = ch
	// Сохранение пира и токена
	peers[peer] = token
	// Сохранение обратно всего в общее хранилище
	s.streams[peer.IdChannel] = peers
	s.log.Debugf("SavePeer. Подключение пира (%s) было успешно сохранено", toString(peer))

	// Рассылка всем, так как подключился новый клиент
	go s.sendPeers(peer.IdChannel)

	return ch, nil
}

// DeletePeer. Удаление пира при отключении
func (s *storage) DeletePeer(peer m.Peer) error {

	s.mx.Lock()
	defer s.mx.Unlock()
	s.log.Debugf("DeletePeer. Пир (%s) удален\n", toString(peer))

	// Получение пиров, который связаны с данном каналом
	peers, ok := s.streams[peer.IdChannel]
	if !ok {
		s.log.Errorf("DeletePeer. Ошибка:%v\n", ErrInvalidIdChannelWhenRemove)
		return ErrInvalidIdChannelWhenRemove
	}

	// Получение токена, который свзан с данным пиром
	token, ok := peers[peer]
	if !ok {
		s.log.Errorf("DeletePeer. Ошибка:%v\n", ErrInvalidPeerWhenRemove)
		return ErrInvalidPeerWhenRemove
	}

	// Пытаемся достать канал, который связан с токеном, закрываем его и удаляем
	ch, ok := s.chans[token]
	if ok {
		close(ch)
		ch = nil
		delete(s.chans, token)
		s.log.Debugf("DeletePeer. Успешно удален канал по токену:%s\n", token)
	}

	// Удаление пира
	delete(peers, peer)
	s.log.Debug("DeletePeer. Успешно удален пир")

	// Обратное сохранение пиров под idch
	s.streams[peer.IdChannel] = peers

	// Рассылка всем об удалении
	go s.sendPeers(peer.IdChannel)

	return nil
}

// sendPeers. Вызывается каждый раз, когда происходят изменения в храненилищах
func (s *storage) sendPeers(idCh m.IdChannel) {
	s.log.Debugf("sendPeers. Произовдится рассылка по idChannel:%s\n", idCh)

	// Пытаемся получить все пиры по idCh
	peers := s.streams[idCh]

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
				ch <- ps
			}(ch)
		}
	}()
}
