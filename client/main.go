package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
)

type Client struct {
	conn *websocket.Conn
	exit chan struct{}
}

var (
	url string
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var (
		user string
		hub  int
	)
	flag.StringVar(&user, "u", "liangweijian", "username")
	flag.IntVar(&hub, "h", 1, "hub index")

	flag.Parse()

	url = fmt.Sprintf("ws://127.0.0.1:8080/chat?username=%s&hub=%d", user, hub)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	c := newClient(conn)

	go c.readMessage()
	go c.writeMessage()

	<-c.exit

}
