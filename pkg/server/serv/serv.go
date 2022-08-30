package serv

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

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
			grpc.StreamInterceptor(s.authStreamInterceptor),
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

func (s *rpcServer) Streaming(stream pb.MyService_StreamingServer) error {

	s.logger.Println("клиент подключился")

	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				log.Println(fmt.Errorf("$Ошибка EOF при чтении, err:=%v", err))
				break
			} else if err != nil {
				s.logger.Println(fmt.Errorf("$Ошибка при чтении сооьбщения, err:=%v", err))
				break
			}

			s.logger.Println("Received Pong message:", in.Pong)
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			s.logger.Println("Клиент отключился, контекст завершен")
			return nil
		case <-s.runningCtx.Done():
			s.logger.Println("Главный контекст завершен")
			return nil
		case <-timer.C:
			err := stream.Send(&pb.Res{
				Ping: true,
			})
			if err == io.EOF {
				log.Println(fmt.Errorf("$Ошибка EOF при писании, err:=%v", err))
				return nil
			}
			if err != nil {
				log.Println(fmt.Errorf("$Ошибка при чтении писании, err:=%v", err))
			}
		}
	}
}


func (s *rpcServer) authStreamInterceptor(srv interface{}, srvStream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	
	log.Println("авторизация пройдена")

	m, ok := metadata.FromIncomingContext(srvStream.Context())
	if !ok {
		return fmt.Errorf("$Ожидались метаданные, err:=%v", err)
	}
	if len(m["authorization"]) != 1 {
		return fmt.Errorf("$Ошибка при авторизации, err:=%v", err)
	}

	if strings.TrimPrefix(m["authorization"][0], "Bearer ") != s.cnf.AuthToken {
		return fmt.Errorf("$Ошибка при авторизации, err:=%v", err)
	}

	return handler(srv, srvStream)
}