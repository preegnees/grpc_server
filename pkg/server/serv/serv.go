package serv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/peer"

	pg "streaming/pkg/database/postgress"
	st "streaming/pkg/database/storage"
	m "streaming/pkg/models"
	pb "streaming/pkg/proto"
)

type rpcServer struct {
	pb.MyServiceServer

	stopCh        chan struct{}
	restartCh     chan struct{}
	doneCh        chan struct{}
	errCh         chan error
	runningCtx    context.Context
	runningCancel context.CancelFunc
	running       bool
	grpcServer    *grpc.Server
	logger        *log.Logger
	cnf           m.CnfServer
	storage       m.IStreamStorage
	pgDb          m.IPostgres
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
		log.Println("Получен сигнал для завершеня работы")
		s.stop()
		return nil
	case err := <-s.errCh:
		s.logger.Printf("Выключение сервера, возникла ошибка")
		s.stop()
		return err
	}
}

func (s *rpcServer) init() {

	s.running = false
	s.logger = log.New(os.Stdout, "grpc: ", log.Ldate|log.Ltime|log.LUTC)
	s.stopCh = make(chan struct{})
	s.restartCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.errCh = make(chan error)
	s.storage = st.NewStorage()
	s.pgDb = pg.New(
		m.ConfPostgres{
			TestMod:  true,
			User:     "postgres",
			Host:     "localhost",
			Password: "1234",
			Port:     "5432",
			Database: "mydb",
			Sslmode:  "disable",
		},
	)
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

		pb.RegisterMyServiceServer(s.grpcServer, s)

		go func() {
			listener, err := net.Listen("tcp", s.cnf.Addr)
			if err != nil {
				s.runningCancel()
				s.errCh <- fmt.Errorf("$Ошибка при прослушивании сети, err:=%v", err)
			}
			s.logger.Printf("Запуск сервера на адресе %s\n", s.cnf.Addr)
			s.running = true
			s.grpcServer.Serve(listener)
		}()

		select {
		case <-s.stopCh:
			s.logger.Println("Выключение сервера, был получен сигнал прерывания")
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
		s.logger.Printf("Превышено время отключения %d секунд, принудительное завершение.\n", s.cnf.ShutdownTimeout)
		s.grpcServer.Stop()
		close(closed)
		closed = nil
	}
	s.logger.Println("Север остановлен")
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
				log.Println(fmt.Errorf("$Ошибка EOF при чтении, err:=%v", err))
				c.deleteThisPeer()
				break
			} else if err != nil {
				log.Println(fmt.Errorf("$Ошибка при чтении сооьбщения, err:=%v", err))
				break
			}

			for p := range c.all {
				if p == c.current {
					continue
				}
				go func(p m.Peer) {
					err := p.GrpcStream.Send(in)
					if err == io.EOF {
						log.Println(fmt.Errorf("$Ошибка EOF при писании, err:=%v", err))
						return
					}
					if err != nil {
						log.Println(fmt.Errorf("$Ошибка при чтении писании, err:=%v", err))
					}
				}(p)
			}
		}
	}()
}

func (s *rpcServer) Streaming(stream pb.MyService_StreamingServer) error {

	peer, err := s.getPeer(stream)
	if err != nil {
		return err
	}

	ch, err := s.storage.SavePeer(*peer)
	if err != nil {
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
	}

	s.logger.Println("клиент подключился", stream)

	newCli.RunForward()

	for {
		select {
		case <-stream.Context().Done():
			s.logger.Println("Клиент отключился, контекст завершен")
			deleteThisPeer()
			return nil
		case <-s.runningCtx.Done():
			s.logger.Println("Главный контекст завершен")
			deleteThisPeer()
			return nil
		}
	}
}

func (r *rpcServer) getPeer(stream pb.MyService_StreamingServer) (*m.Peer, error) {

	md, ok := metadata.FromIncomingContext(stream.Context())
	fmt.Println(md)
	if !ok {
		return nil, errors.New("Error with metadata")
	}

	peer, ok := peer.FromContext(stream.Context())
	if !ok {
		return nil, errors.New("Error with peer")
	}
	ip := peer.Addr.String()

	i := m.IdChannel(md["idchannel"][0])
	n := m.Name(md["name"][0])
	a := md["allowednames"][0]
	g := stream
	if i == "" || len(i) < 8 {
		return nil, errors.New("Invalid IdChannel, mast be not empty and len > 8")
	} else if n == "" {
		return nil, errors.New("Invalid Name, mast be not empty")
	}
 
	r.pgDb.Save(m.Connection{
		IP:           ip,
		Name:         n,
		IdChannel:    i,
		AllowedNames: a,
		Time:         time.Now(),
	})

	return &m.Peer{
		IdChannel:    i,
		Name:         n,
		AllowedNames: a,
		GrpcStream:   g,
	}, nil
}
