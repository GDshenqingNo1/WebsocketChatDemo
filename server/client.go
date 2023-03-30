package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type msg struct {
	Username string `json:"username"`
	Time     string `json:"time"`
	Message  string `json:"message"`
}

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan *msg
	username string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	defaultHandle := c.conn.CloseHandler()
	c.conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("receive close message code:%d text:%s", code, text)
		c.hub.unregister <- c
		c.hub.broadcast <- &msg{
			Username: c.username,
			Time:     time.Now().Format(time.RFC3339),
			Message:  fmt.Sprintf("%s离开聊天室", c.username),
		}
		return defaultHandle(code, text)
	})
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("error:%v", err)
			}

			//检查是不是客户端断了
			if netErr, ok := err.(*net.OpError); ok && netErr.Err.Error() == "wsarecv: An existing connection was forcibly closed by the remote host." {
				c.hub.broadcast <- &msg{
					Username: c.username,
					Time:     time.Now().Format(time.RFC3339),
					Message:  fmt.Sprintf("%s离开聊天室", c.username),
				}

			} else {
				log.Println(err)
			}
			break
		}

		var message = &msg{
			Username: c.username,
			Time:     time.Now().Format(time.RFC3339),
			Message:  string(p),
		}
		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			//判断send通道是不是已经被关闭
			if !ok {
				err := c.conn.WriteMessage(websocket.CloseMessage, nil)
				if err != nil {
					//判断之前是不是已经发过CloseMessage了
					if errors.Is(err, websocket.ErrCloseSent) {
						return
					}

					//判断连接是不是已经被关闭了
					if netErr, ok := err.(*net.OpError); ok && netErr.Err.Error() == "use of closed network connection" {
						return
					}

					log.Println(err)
				}
				return
			}
			c.conn.WriteJSON(message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serve(c *gin.Context) {
	username := c.Query("username")
	hubId, err := strconv.Atoi(c.Query("hub"))
	if username == "" || err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "username or group cannot be null",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	hub := getHub(hubId)

	client := &Client{username: username, hub: hub, conn: conn, send: make(chan *msg, 256)}
	client.hub.register <- client
	client.hub.broadcast <- &msg{
		Username: "system",
		Time:     time.Now().Format(time.RFC3339),
		Message:  fmt.Sprintf("%s进入聊天室", username),
	}

	go client.writePump()
	go client.readPump()

}
