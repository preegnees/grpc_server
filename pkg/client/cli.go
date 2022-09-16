package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	m "streaming/pkg/models"
	pb "streaming/pkg/proto"
)

var ErrListenGops = errors.New("$Ошибка при прослкшивании gops")
var ErrConnectionToServer = errors.New("$Ошибка при подключении")

type rpcClient struct {
	pb.StreamingServiceClient

	connection *grpc.ClientConn
	logger     *log.Logger
	errCh      chan error
	cnf        m.CnfClient
}

var _ m.ICli = (*rpcClient)(nil)

func New() m.ICli {

	return &rpcClient{}
}

func (c *rpcClient) Run(cnf m.CnfClient) error {

	c.cnf = cnf

	c.init()
	err := c.connect()
	if err != nil {
		return fmt.Errorf("%w, err:=%v", ErrConnectionToServer, err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh)

	select {
	case <-sigCh:
		c.logger.Println("Получен сигнал об окончании работы")
		err = c.disconnect()
		if err != nil {
			return fmt.Errorf("$Ошибка при отключении от сервера, err:=%v", err)
		}
		return nil
	case err := <-c.errCh:
		ed := c.disconnect()
		if ed != nil {
			ed = fmt.Errorf("$Ошибка при отключении от сервера, err:=%v", ed)
		}
		return fmt.Errorf("$Произошла ошибка во время работы, err:=%v, %v", err, ed)
	}
}

func (c *rpcClient) Stop() {
	c.disconnect()
}

func (c *rpcClient) init() {

	c.logger = log.New(os.Stdout, "grpc: ", log.Ldate|log.Ltime|log.LUTC)
	c.errCh = make(chan error)
}

type Cli struct {
	IdChannel    string
	Name         string
	AllowedNames string
}

func (a *Cli) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {

	return map[string]string{
		"IdChannel":    a.IdChannel,
		"Name":         a.Name,
		"AllowedNames": a.AllowedNames,
	}, nil
}

func (a *Cli) RequireTransportSecurity() bool {

	return true
}

func (c *rpcClient) connect() (err error) {

	connOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})),
		grpc.WithPerRPCCredentials(&Cli{
			IdChannel:    c.cnf.IdChannel,
			Name:         c.cnf.Name,
			AllowedNames: c.cnf.AllowedNames,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: time.Duration(c.cnf.RequestTimeout) * time.Second,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Duration(c.cnf.KeepaliveInterval) * time.Second,
			Timeout: time.Duration(c.cnf.RequestTimeout) * time.Second,
		}),
	}

	c.connection, err = grpc.Dial(c.cnf.Addr, connOpts...)
	if err != nil {
		return err
	}

	c.StreamingServiceClient = pb.NewStreamingServiceClient(c.connection)

	go func() {
		for c.connection != nil {
			c.logger.Println("Подключение к серверу")
			c.startStream()
			time.Sleep(1 * time.Second)
		}
	}()

	return nil
}

func (c *rpcClient) disconnect() error {

	conn := c.connection
	c.connection = nil
	err := conn.Close()
	return err
}

func (c *rpcClient) startStream() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := c.Streaming(ctx, grpc.WaitForReady(true))
	if err != nil {
		c.errCh <- fmt.Errorf("Error while connecting to the Channel stream:%v", err)
		return
	}
	defer stream.CloseSend()
	c.logger.Println("Канал подключен")

	timer := time.NewTicker(500 * time.Millisecond)
	defer timer.Stop()

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				log.Println(fmt.Errorf("$Ошибка EOF при чтении, err:=%v", err))
				cancel()
				break
			}
			if err != nil {
				log.Println(fmt.Errorf("$Ошибка при чтении сооьбщения, err:=%v", err))
				break
			}

			c.logger.Printf("Received Ping message: %s\n", in.Bs)
		}
	}()

	for {
		select {
		case <-timer.C:
			err := stream.Send(&pb.Message{
				Bs: []byte(fmt.Sprintf("hello world, this is %s", c.cnf.Name)),
			})
			if err == io.EOF {
				log.Println(fmt.Errorf("$Ошибка EOF при писании, err:=%v", err))
				return
			}
			if err != nil {
				c.errCh <- fmt.Errorf("$Ошибка при чтении писании, err:=%v", err)
			}
		case <-ctx.Done():
			c.logger.Println("Канал закрыт")
			return
		}
	}
}
