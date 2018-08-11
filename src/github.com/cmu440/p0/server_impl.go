package p0

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

// MAXN limits the max messages a client can write.
const MAXN = 100

type multiEchoServer struct {
	listener       net.Listener
	currentClients []*client
	newMessage     chan []byte
	newConnection  chan net.Conn
	deadClient     chan *client
	countClients   chan int
	clientCount    chan int
	quitMain       chan int
	quitAccept     chan int
}

type client struct {
	conn         net.Conn
	messageQueue chan []byte
	quitRead     chan int
	quitWrite    chan int
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	s := &multiEchoServer{
		nil,
		nil,
		make(chan []byte),
		make(chan net.Conn),
		make(chan *client),
		make(chan int),
		make(chan int),
		make(chan int),
		make(chan int)}

	return s
}

func (mes *multiEchoServer) Start(port int) error {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	mes.listener = listener
	go runServer(mes)
	go acceptConnection(mes)
	return nil
}

func runServer(mes *multiEchoServer) {
	for {
		select {
		case newMessage := <-mes.newMessage:
			for _, c := range mes.currentClients {
				if len(c.messageQueue) == MAXN {
					<-c.messageQueue
				}
				c.messageQueue <- newMessage
			}
		case newConnection := <-mes.newConnection:
			c := &client{
				newConnection,
				make(chan []byte, MAXN),
				make(chan int),
				make(chan int)}
			mes.currentClients = append(mes.currentClients, c)
			go handleRead(mes, c)
			go handleWrite(c)

		case deadClient := <-mes.deadClient:
			for i, c := range mes.currentClients {
				if c == deadClient {
					mes.currentClients =
						append(mes.currentClients[:i], mes.currentClients[i+1:]...)
					break
				}
			}
		case <-mes.countClients:
			mes.clientCount <- len(mes.currentClients)

		case <-mes.quitMain:
			for _, c := range mes.currentClients {
				c.conn.Close()
				c.quitWrite <- 0
				c.quitRead <- 0
			}
			return
		}
	}

}

func acceptConnection(mes *multiEchoServer) {
	for {
		select {
		case <-mes.quitAccept:
			return
		default:
			conn, err := mes.listener.Accept()
			if err == nil {
				mes.newConnection <- conn
			}
		}
	}
}

func (mes *multiEchoServer) Count() int {
	mes.countClients <- 0
	return <-mes.clientCount
}

func (mes *multiEchoServer) Close() {
	mes.listener.Close()
	mes.quitAccept <- 0
	mes.quitMain <- 0
}

func handleRead(mes *multiEchoServer, c *client) {
	clientReader := bufio.NewReader(c.conn)

	for {
		select {
		case <-c.quitRead:
			return
		default:
			message, err := clientReader.ReadBytes('\n')

			if err == io.EOF {
				mes.deadClient <- c
			} else if err != nil {
				return
			} else {
				mes.newMessage <- message
			}
		}
	}
}

func handleWrite(c *client) {
	for {
		select {
		case <-c.quitWrite:
			return
		case message := <-c.messageQueue:
			c.conn.Write(message)
		}
	}
}
