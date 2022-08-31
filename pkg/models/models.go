package models

import (
	pb "streaming/pkg/proto"
)

// CnfServer ...
type CnfServer struct {
	Addr            string
	AuthToken       string
	Restart         bool
	ShutdownTimeout int
	CertPem         string
	KeyPem          string
}

// CnfClient ...
type CnfClient struct {
	Addr              string
	AuthToken         string
	RequestTimeout    int
	KeepaliveInterval int
	Reconnect         bool
	ReconnectTimeout  int
}

// IServ ...
type IServ interface {
	Run(CnfServer) error
}

// ICli ...
type ICli interface {
	Run(CnfClient) error
}

type Peer struct {
	Name      string
	Connected bool
	Stream    pb.MyService_StreamingServer
}

type Stream struct {
	IdChannel string
	Peers     []Peer
}

// IMemStorage ...
type IStreamStorage interface {
	SaveStream(strm Stream) (string, error)
	SavePeer(idChannel string, peer Peer) error
	SetFiledConnected(idChannel string, peer Peer, is bool) error
	GetStreams() (strm chan<- Stream)
}

// IAccessStorage ...
type IAccessStorage interface {
	Check(string) (bool, error)
	SaveNewName(string) (bool, error)
}

// IStorage ...
type IStorage interface {
	IAccessStorage
	IStreamStorage
}
