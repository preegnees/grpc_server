package models

import (
	pb "streaming/pkg/proto"
)

// ---> Conf

// CnfServer. Конифгурация для сервера
type CnfServer struct {
	Addr            string
	ShutdownTimeout int
	CertPem         string
	KeyPem          string
}

// CnfClient. Конфигурация для клиента
type CnfClient struct {
	Addr              string
	RequestTimeout    int
	KeepaliveInterval int
	Reconnect         bool
	ReconnectTimeout  int
	IdChannel         string
	Name              string
	AllowedNames      string
}

// IServ. Интерфейс для сервера
type IServ interface {
	Run(CnfServer) error
	Stop()
}

// ICli. Интерфейс для клиента
type ICli interface {
	Run(CnfClient) error
	Stop()
}

// ---> MyMemStorage

// IdChannel - канала, который связывет пиры
type IdChannel string

// Name - имя клиента
type Name string

// Token - токен, который связывает каналы и пиры в хранилище
type Token string

/*
Peer. Пир - это подключение, которое имеет:
IdChannel: id канала, с которым он связан;
Name: имя, которое имеет пир;
AllowedNames: разрешенные имена, с которым хочет соединится пир. Перечислять нужно через ","
GrpcStream=стрим gprc;
*/
type Peer struct {
	IdChannel    IdChannel
	Name         Name
	AllowedNames string
	GrpcStream   pb.StreamingService_StreamingServer
}

/*
IMemStorage. Интерфейс для взаимодействия с базой данных, которая хронит стримы.
SavePeer: сохранение пира при подключении;
DeletePeer: удаление пира при отключении;
*/
type IStreamStorage interface {
	SavePeer(peer Peer) (<-chan map[Peer]struct{}, error)
	DeletePeer(peer Peer) error
}