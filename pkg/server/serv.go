package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	st "streaming/pkg/database/storage"
	m "streaming/pkg/models"
	pb "streaming/pkg/proto"
	logger "streaming/pkg/logger"
	e "streaming/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	logrus "github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
)

type rpcServer struct {
	pb.StreamingServiceServer

	stopCh        chan struct{}
	restartCh     chan struct{}
	doneCh        chan struct{}
	errCh         chan error
	runningCtx    context.Context
	runningCancel context.CancelFunc
	running       bool
	grpcServer    *grpc.Server
	log           *logrus.Logger
	cnf           m.CnfServer
	storage       m.IStreamStorage
}

var _ m.IServ = (*rpcServer)(nil)

func New() m.IServ {

	return &rpcServer{}
}

func (s *rpcServer) Run(cnf m.CnfServer) error {

	s.cnf = cnf

	s.init()
	go s.start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	select {
	case <-sigCh:
		s.log.Info("Получен сигнал для завершеня работы")
		s.stop()
		return nil
	case err := <-s.errCh:
		s.log.Errorf("Выключение сервера, возникла ошибка, err := %v", err)
		s.stop()
		return err
	}
}

func (s *rpcServer) Stop() {
	s.log.Info("Остановка сервера")
	s.gracefulStop()
}

func (s *rpcServer) init() {

	s.running = false
	s.stopCh = make(chan struct{})
	s.restartCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.errCh = make(chan error)
	s.storage = st.NewStorage(logger.NewLogger(s.cnf.Debug, s.cnf.StreamStoragePathLog))
	s.log = logger.NewLogger(s.cnf.Debug, s.cnf.ServerPathLog)
}

func (s *rpcServer) start() {

	for {
		s.runningCtx, s.runningCancel = context.WithCancel(context.Background())
		creds, err := credentials.NewServerTLSFromFile(s.cnf.CertPem, s.cnf.KeyPem)
		if err != nil {
			s.errCh <- fmt.Errorf("$Ошибка при добавлении TLS, err:=%v", err)
		}

		s.grpcServer = grpc.NewServer(
			grpc.Creds(creds),
		)

		pb.RegisterStreamingServiceServer(s.grpcServer, s)

		go func() {
			listener, err := net.Listen("tcp", s.cnf.Addr)
			if err != nil {
				s.runningCancel()
				s.errCh <- fmt.Errorf("$Ошибка при прослушивании сети, err:=%v", err)
			}
			s.log.Infof("Запуск сервера на адресе %s", s.cnf.Addr)
			s.running = true
			s.grpcServer.Serve(listener)
		}()

		select {
		case <-s.stopCh:
			s.log.Info("Выключение сервера, был получен сигнал прерывания")
			s.gracefulStop()
			s.running = false
			s.doneCh <- struct{}{}
			return
		}
	}
}

func (s *rpcServer) gracefulStop() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cnf.ShutdownTimeout)*time.Second)
	defer cancel()

	s.runningCancel()

	closed := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		if closed != nil {
			select {
			case closed <- struct{}{}:
			default:
			}
		}
	}()

	select {
	case <-closed:
		close(closed)
	case <-ctx.Done():
		s.log.Debugf("Превышено время отключения %d секунд, принудительное завершение.", s.cnf.ShutdownTimeout)
		s.grpcServer.Stop()
		close(closed)
		closed = nil
	}
	s.log.Info("Север остановлен")
}

func (s *rpcServer) stop() {

	if s.running {
		s.stopCh <- struct{}{}
		<-s.doneCh
	}
}

type cli struct {
	current        m.Peer
	all            map[m.Peer]struct{}
	ch             <-chan map[m.Peer]struct{}
	deleteThisPeer func() error
	log            *logrus.Logger
}

func (c *cli) RunForward() {

	go func() {
		for {
			select {
			case newAll, ok := <-c.ch:
				if !ok {
					return
				}
				c.all = newAll
			}
		}
	}()

	go func() {
		for {
			in, err := c.current.GrpcStream.Recv()
			if err == io.EOF {
				c.log.Warn(fmt.Errorf("$Ошибка EOF при чтении, err:=%v", err))
				c.deleteThisPeer()
				break
			} else if err != nil {
				c.log.Warn(fmt.Errorf("$Ошибка при чтении сооьбщения, err:=%v", err))
				break
			}

			for p := range c.all {
				if p == c.current {
					continue
				}
				go func(p m.Peer) {
					err := p.GrpcStream.Send(in)
					if err == io.EOF {
						c.log.Warn(fmt.Errorf("$Ошибка EOF при писании, err:=%v", err))
						return
					}
					if err != nil {
						c.log.Warn(fmt.Errorf("$Ошибка при чтении писании, err:=%v", err))
					}
				}(p)
			}
		}
	}()
}

func (s *rpcServer) Streaming(stream pb.StreamingService_StreamingServer) error {

	peer, err := s.getPeer(stream)
	defer s.log.WithFields(logrus.Fields{"clientInfo": logger.CliConn{Ip: "__ip__", Peer: *peer}}).Debug("Клиент отключился")
	if err != nil {
		s.log.Errorf("Ошибка при получении пира (getPeer). Ошибка:%v", err)
		return err
	}

	ch, err := s.storage.SavePeer(*peer)
	if err != nil {
		s.log.Errorf("Ошибка при сохранении пира (SavePeer). Ошибка:%v", err)
		return err
	}

	deleteThisPeer := func() error {
		return s.storage.DeletePeer(*peer)
	}
	newCli := cli{
		current:        *peer,
		all:            make(map[m.Peer]struct{}),
		ch:             ch,
		deleteThisPeer: deleteThisPeer,
		log:            s.log,
	}

	s.log.Infof("клиент подключился, stream:%v", stream)

	newCli.RunForward()

	for {
		select {
		case <-stream.Context().Done():
			s.log.Info("Клиент отключился, контекст завершен")
			deleteThisPeer()
			return nil
		case <-s.runningCtx.Done():
			s.log.Info("Главный контекст завершен")
			deleteThisPeer()
			return nil
		}
	}
}

func (s *rpcServer) getPeer(stream pb.StreamingService_StreamingServer) (*m.Peer, error) {

	md, ok := metadata.FromIncomingContext(stream.Context())
	fmt.Println(md)
	if !ok {
		s.log.Errorf("getPeer. Ошибка:%v", e.ErrGetMetadata)
		return nil, e.ErrGetMetadata
	}

	peer, ok := peer.FromContext(stream.Context())
	var ip string = ""
	if !ok {
		s.log.Errorf("Ошибка при получении ip из библиотека Peer")
		ip = "err no ip"
	} else {
		ip = peer.Addr.String()
	}

	if len(md["idchannel"]) != 1 || len(md["name"]) != 1 || len(md["allowednames"]) != 1 {
		return nil, e.ErrNoMetadata
	}

	i := m.IdChannel(md["idchannel"][0])
	n := m.Name(md["name"][0])
	a := md["allowednames"][0]
	g := stream
	if i == "" || len(i) < 8 {
		s.log.Errorf("getPeer. Ошибка:%v", e.ErrInvalidIdChannel)
		return nil, e.ErrInvalidIdChannel
	} else if n == "" {
		s.log.Errorf("getPeer. Ошибка:%v", e.ErrInvalidIdName)
		return nil, e.ErrInvalidIdName
	}

	cc := logger.CliConn{
		Ip: ip,
		Peer: m.Peer{
			IdChannel: i,
			Name: n,
			AllowedNames: a,
			GrpcStream: g,
		},
	}

	s.log.WithFields(logrus.Fields{"clientInfo": cc}).Info("Клиент подключился")

	return &m.Peer{
		IdChannel:    i,
		Name:         n,
		AllowedNames: a,
		GrpcStream:   g,
	}, nil
}
