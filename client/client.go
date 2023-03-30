package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"os"
	"strings"
)

func (c *Client) readMessage() {
	defer func() {
		close(c.exit)
	}()
	for {
		_, p, e := c.conn.ReadMessage()
		if e != nil {
			//判断是不是连接不正常关闭
			if websocket.IsUnexpectedCloseError(e, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println(e)
			}

			//判断是不是与服务端断开连接
			if e, ok := e.(net.Error); ok {
				log.Printf("network error : %v", e)
			}

			//断线重连
			for count := 0; count < 5; count++ {
				c.conn, _, e = websocket.DefaultDialer.Dial(url, nil)
				if e != nil {
					log.Println(e)
					continue
				}
				break
			}
			if e != nil {
				return

			}

		} else {
			fmt.Println(string(p))
		}
	}
}

func (c *Client) writeMessage() {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-c.exit:
			return
		default:
			cmd, _ := reader.ReadString('\n')
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {

				if string(cmd) == "q" || string(cmd) == "Q" {
					c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
					c.exit <- struct{}{}
				}

				err := c.conn.WriteMessage(websocket.TextMessage, []byte(cmd))
				if err != nil {
					//	if errors.Is(err, websocket.ErrCloseSent) {
					//		return
					//	}
					log.Fatalln("write message error", err)
				}
			}
		}
	}
}

func newClient(conn *websocket.Conn) *Client {

	var c = &Client{
		conn: conn,
		exit: make(chan struct{}),
	}

	//设置CustomCloseHandler
	defaultHandler := conn.CloseHandler()
	customHandler := func(code int, text string) error {
		log.Printf("receive close message code:%d text:%s", code, text)
		c.exit <- struct{}{}
		conn.Close()
		return defaultHandler(code, text)
	}
	c.conn.SetCloseHandler(customHandler)

	//设置pingHandler
	conn.SetPingHandler(func(appData string) error {
		log.Println("get ping message")
		//conn.WriteMessage(websocket.PongMessage, nil)
		return nil
	})

	return c
}
