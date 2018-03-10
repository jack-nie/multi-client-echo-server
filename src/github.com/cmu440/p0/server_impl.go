// Implementation of a MultiEchoServer. Students should write their code in this file.
package p0

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

type multiEchoServer struct {
	ls          net.Listener
	clientCount int
	clients     chan map[string]multiEchoClient
	msg         chan []byte
}

type multiEchoClient struct {
	msg       chan []byte
	conn      net.Conn
	closeChan chan struct{}
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	s := &multiEchoServer{
		ls:          nil,
		clientCount: 0,
		clients:     make(chan map[string]multiEchoClient, 1),
		msg:         make(chan []byte, 100),
	}
	s.clients <- make(map[string]multiEchoClient, 1)
	return s
}

func (mes *multiEchoServer) Start(port int) error {
	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Failed to listen on post: ", port)
	}
	mes.ls = listen

	go func() {
		for {
			conn, err := mes.ls.Accept()
			if err != nil {
				return
			}
			addClient(mes, conn)
		}
	}()

	go func() {
		for {
			msg, ok := <-mes.msg
			if !ok {
				return
			}
			cli := <-mes.clients
			for _, client := range cli {
				select {
				case client.msg <- msg:
					break
				default:
					break
				}
			}
			mes.clients <- cli
		}
	}()
	return err
}

func (mes *multiEchoServer) Close() {
	cli := <-mes.clients
	for _, client := range cli {
		client.closeChan <- struct{}{}
	}
	mes.clients <- cli
	mes.ls.Close()
}

func (mes *multiEchoServer) Count() int {
	return mes.clientCount
}

func addClient(mes *multiEchoServer, conn net.Conn) {
	client := multiEchoClient{
		msg:       make(chan []byte, 100),
		conn:      conn,
		closeChan: make(chan struct{}),
	}

	cli := <-mes.clients
	cli[conn.RemoteAddr().String()] = client
	mes.clients <- cli
	mes.clientCount++
	go func() {
		for {
			select {
			case msg, ok := <-client.msg:
				if !ok {
					return
				}
				_, err := conn.Write(msg)
				if err != nil {
					return
				}
			case <-client.closeChan:
				client.conn.Close()
				removeClient(mes, conn)
			}
		}
	}()

	go func() {
		content := bufio.NewReader(conn)
		for {
			line, err := content.ReadBytes('\n')
			if err != nil {
				client.closeChan <- struct{}{}
				return
			}
			mes.msg <- line
		}
	}()
}

func removeClient(mes *multiEchoServer, conn net.Conn) {
	cli := <-mes.clients
	delete(cli, conn.RemoteAddr().String())
	mes.clients <- cli
	mes.clientCount--
}
