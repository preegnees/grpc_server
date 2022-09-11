package models

import (
	"time"

	pb "streaming/pkg/proto"
)

// ---> Conf

// CnfServer. Конифгурация для сервера
type CnfServer struct {
	Addr            string
	AuthToken       string
	Restart         bool
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
}

// ICli. Интерфейс для клиента
type ICli interface {
	Run(CnfClient) error
}

// ---> MyMemStorage

// IdChannel - канала, который связывет пиры
type IdChannel string

// Name - имя клиента
type Name string

// Token - токен для связки в хранилище каналы и пиры
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
	GrpcStream   pb.MyService_StreamingServer
}

/*
IMemStorage. Интерфейс для взаимодействия с базой данных.
SavePeer: сохранение пира;
DeletePeer: удаление пира при отключении;
*/
type IStreamStorage interface {
	SavePeer(peer Peer) (<-chan map[Peer]struct{}, error)
	DeletePeer(peer Peer) error
}

// ---> Postgress

type ConfPostgres struct {
	TestMod  bool
	User     string
	Host     string
	Password string
	Port     string
	Database string
	Sslmode  string
}

type Connection struct {
	ID           uint `gorm:"primaryKey"`
	IP           string
	Name         Name
	IdChannel    IdChannel
	AllowedNames string
	Time         time.Time
}

type IPostgres interface {
	Save(Connection) error
	GetConnsUseName(Name) ([]Connection, error)
	GetConnsUseIdChannel(IdChannel) ([]Connection, error)
}
