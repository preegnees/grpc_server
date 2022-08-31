package serv

import (
	"context"
	"fmt"
	// "io"
	"bytes"
	"encoding/gob"
	//
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

var streams map[pb.MyService_StreamingServer]struct{} = make(map[pb.MyService_StreamingServer]struct{}, 0)
var access map[string]struct{} = map[string]struct{}{
	"1": struct{}{},
	"2": struct{}{},
}

func (s *rpcServer) Streaming(stream pb.MyService_StreamingServer) error {

	//
	// gob.Register(map[pb.MyService_StreamingServer]interface{}{})
	gob.Register(map[pb.MyService_StreamingServer]interface{}{})
	var buff bytes.Buffer

	// var in map[pb.MyService_StreamingServer]interface{} = map[pb.MyService_StreamingServer]interface{}{stream:struct{}{}}
	// var out map[pb.MyService_StreamingServer]interface{} = make(map[pb.MyService_StreamingServer]interface{})
	var in pb.MyService_StreamingServer = stream
	var out pb.MyService_StreamingServer
	// // Кодирование значения
	// var in map[grpc.ServerStream]interface{} = map[grpc.ServerStream]interface{} {
	// 	stream: struct{}{},
	// }
	log.Println(out)
	enc := gob.NewEncoder(&buff)
	dec := gob.NewDecoder(&buff)
	errEnc := enc.Encode(&in)
	log.Println("errEnc:", errEnc)
	errDec := dec.Decode(&out)
	log.Println("errDec:", errDec)
	log.Println(out)
	// err := enc.Encode(in)
	// log.Println(err)
	// fmt.Printf("%X\n", buff.Bytes())
 
	// // Декодирование значения
	// var out Test = Test{}
	// dec := gob.NewDecoder(&buff)
	// err = dec.Decode(&out)
	// log.Println(err)
	// fmt.Println(out)
	return nil
	//

	s.logger.Println("клиент подключился", stream)

	streams[stream] = struct{}{}
	log.Println("состояние стримов после подключения:", streams)

	for {
		if len(streams) == 2 {
			break
		}
		select {
		case <- stream.Context().Done():
			log.Println("клиент отключился", stream)
			delete(streams, stream)
			return nil
		}
	}

	strms := make([]pb.MyService_StreamingServer, 0)

	for k := range streams {
		strms = append(strms, k)
	}
	
	retCh := make(chan error)
	errStr1 := make(chan pb.MyService_StreamingServer)
	errStr2 := make(chan pb.MyService_StreamingServer)

	go s.logic(errStr1, errStr1, retCh, strms[0], strms[1])
	go s.logic(errStr2, errStr1, retCh, strms[1], strms[0])
	
	// тогда нужно передавать отключение для двух каналов

	select{
	case err:= <-retCh:
		return err
	case str := <-errStr1:
		delete(streams, str)
		log.Println("состояние стримов после удаления:", streams)
		s.logger.Println("stream:", str)
		return fmt.Errorf("%v", str)
	case str := <-errStr2:
		delete(streams, str)
		log.Println("состояние стримов после удаления:", streams)
		s.logger.Println("stream:", str)
		return fmt.Errorf("%v", str)
	}
}

func (s *rpcServer) logic(errStr1, errStr2 chan pb.MyService_StreamingServer, retCh chan error, stream1, stream2 pb.MyService_StreamingServer) error {
	
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-stream1.Context().Done():
			s.logger.Println("Клиент отключился, контекст завершен")
			errStr1 <- stream1
			// stream2.Context().Err()
			// retCh <- nil
			return nil
		case <-stream2.Context().Done():
			s.logger.Println("Клиент отключился, контекст завершен")
			errStr2 <- stream2
			// stream1.Context().Err()
			// retCh <- nil
			return nil
		case <-s.runningCtx.Done():
			s.logger.Println("Главный контекст завершен")
			// retCh <- nil
			return nil
		case <-timer.C:

			// in, err := stream1.Recv()
			// if err == io.EOF {
			// 	log.Println(fmt.Errorf("$Ошибка EOF при чтении, err:=%v", err))
			// 	// retCh <- nil
			// 	return nil
			// } else if err != nil {
			// 	s.logger.Println(fmt.Errorf("$Ошибка при чтении сооьбщения, err:=%v", err))
			// }
			
			// err = stream2.Send(&pb.Res{
			// 	Ping: in.Pong,
			// })
			// if err == io.EOF {
			// 	log.Println(fmt.Errorf("$Ошибка EOF при писании, err:=%v", err))
			// 	// retCh <- nil
			// 	return nil
			// }
			// if err != nil {
			// 	log.Println(fmt.Errorf("$Ошибка при чтении писании, err:=%v", err))
			// }
		}
	}
}


// s.logger.Println("клиент подключился")

// 	timer := time.NewTicker(2 * time.Second)
// 	defer timer.Stop()

// 	go func() {
// 		for {
// 			in, err := stream.Recv()
// 			if err == io.EOF {
// 				log.Println(fmt.Errorf("$Ошибка EOF при чтении, err:=%v", err))
// 				break
// 			} else if err != nil {
// 				s.logger.Println(fmt.Errorf("$Ошибка при чтении сооьбщения, err:=%v", err))
// 				break
// 			}

// 			s.logger.Println("Received Pong message:", in.Pong)
// 		}
// 	}()

// 	for {
// 		select {
// 		case <-stream.Context().Done():
// 			s.logger.Println("Клиент отключился, контекст завершен")
// 			retCh <- nil
// 			return nil
// 		case <-s.runningCtx.Done():
// 			s.logger.Println("Главный контекст завершен")
// 			retCh <- nil
// 			return nil
// 		case <-timer.C:
// 			err := stream.Send(&pb.Res{
// 				Ping: true,
// 			})
// 			if err == io.EOF {
// 				log.Println(fmt.Errorf("$Ошибка EOF при писании, err:=%v", err))
// 				retCh <- nil
// 				return nil
// 			}
// 			if err != nil {
// 				log.Println(fmt.Errorf("$Ошибка при чтении писании, err:=%v", err))
// 			}
// 		}
// 	}

func (s *rpcServer) authStreamInterceptor(srv interface{}, srvStream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {

	log.Println("авторизация пройдена")

	m, ok := metadata.FromIncomingContext(srvStream.Context())
	if !ok {
		return fmt.Errorf("$Ожидались метаданные, err:=%v", err)
	}
	if len(m["authorization"]) != 1 {
		return fmt.Errorf("$Ошибка при авторизации, err:=%v", err)
	}

	_, ok = access[m["authorization"][0]]
	if !ok {
		return fmt.Errorf("$Ошибка при авторизации, err:=%v", err)
	}
	return handler(srv, srvStream)
}
