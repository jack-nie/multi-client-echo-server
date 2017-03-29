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
	clients     map[string]multiEchoClient
	msg         chan []byte
}

type multiEchoClient struct {
	msg  chan []byte
	conn net.Conn
}

// New creates and returns (but does not start) a new MultiEchoServer.
func New() MultiEchoServer {
	return &multiEchoServer{
		ls:          nil,
		clientCount: 0,
		clients:     make(map[string]multiEchoClient),
		msg:         make(chan []byte),
	}
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
			if err == nil {
				addClient(mes, conn)
			}
		}
	}()

	go func() {
		for {
			msg := <-mes.msg
			for _, client := range mes.clients {
				if len(client.msg) < 100 {
					client.msg <- msg
				}
			}
		}
	}()
	return err
}

func (mes *multiEchoServer) Close() {
	for _, client := range mes.clients {
		client.conn.Close()
		mes.clientCount--
	}
	mes.ls.Close()
}

func (mes *multiEchoServer) Count() int {
	return mes.clientCount
}

func addClient(mes *multiEchoServer, conn net.Conn) {
	client := multiEchoClient{
		msg:  make(chan []byte, 100),
		conn: conn,
	}

	mes.clients[conn.RemoteAddr().String()] = client
	mes.clientCount++
	go func() {
		for {
			msg := <-client.msg
			_, err := conn.Write(msg)
			if err != nil {
				removeClient(mes, conn)
				return
			}
		}
	}()

	go func() {
		content := bufio.NewReader(conn)
		for {
			line, err := content.ReadBytes('\n')
			if err != nil {
				removeClient(mes, conn)
				return
			}
			mes.msg <- line
		}
	}()
}

func removeClient(mes *multiEchoServer, conn net.Conn) {
	delete(mes.clients, conn.RemoteAddr().String())
	mes.clientCount--
}
