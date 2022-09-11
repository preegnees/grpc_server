package storage

// import (
// 	"fmt"
// 	"os"
// 	"sync"
// 	"time"
// 	"testing"

// 	m "streaming/pkg/models"

// 	"go.uber.org/goleak"
// )

// var stor m.IStreamStorage

// func TestMain(m *testing.M) {

// 	fmt.Println("Инициализация")
// 	stor = NewStorage()
// 	fmt.Println("Сторадж инициализирван:", stor)
// 	exitEval := m.Run()
// 	fmt.Println("Остановка")
// 	os.Exit(exitEval)
// }

// Тест скопирован, его не нужно запускать
// func TestSavePeersAndGetPeers(t *testing.T) {

	// defer goleak.VerifyNone(t)
	// var idCh1 m.IdChannel = "1"
	// var idCh2 m.IdChannel = "2"

	// data := []struct {
		// Name string
		// Peer m.Peer
	// }{
		// {"1", m.Peer{IdChannel: idCh1, Name: "N1", GrpcStream: "test1"}},
		// {"2", m.Peer{IdChannel: idCh1, Name: "N2", GrpcStream: "test2"}},
		// {"3", m.Peer{IdChannel: idCh1, Name: "N3", GrpcStream: "test3"}},
		// {"4", m.Peer{IdChannel: idCh2, Name: "N4", GrpcStream: "test4"}},
	// }

	// chs := make([]<-chan map[m.Peer]struct{}, 0)

// 	for _, d := range data {
// 		ch, _ := stor.SavePeer(d.Peer)
// 		chs = append(chs, ch)
// 	}
	
// 	if len(chs) == 0 {
// 		fmt.Println("Длинна массива chs = 0")
// 	}

// 	count := 0
// 	var mx sync.Mutex
// 	incr := func () {
// 		mx.Lock()
// 		defer mx.Unlock()
// 		count++
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(len(chs))

// 	for i, ch := range chs {
// 		go func (i int, ch <-chan map[m.Peer]struct{})  {
// 			for {
// 				select {
// 				case val, ok := <-ch:
// 					if ok {
// 						incr()
// 						fmt.Println("index=", i, ", val=", val)	
// 						continue
// 					}
// 					wg.Done()
// 					fmt.Println("index=", i, ", closed")
// 					return
// 				}
// 			}
// 		}(i, ch)
// 	}

// 	for _, d := range data {
// 		time.Sleep(1 * time.Second)
// 		stor.DeletePeer(d.Peer)
// 		fmt.Println(d.Peer)
// 	}
// 	wg.Wait()

// 	fmt.Println("---->", count)
// }