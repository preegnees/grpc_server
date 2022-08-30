package models

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
