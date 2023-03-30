package main

import (
	"sync"
)

type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan *msg
	clients    map[*Client]bool
	mu         sync.Mutex
}

func getHub(hubId int) *Hub {
	if v, ok := hubs.Load(hubId); ok {
		return v.(*Hub)
	} else {
		var newHub = &Hub{
			broadcast:  make(chan *msg),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			clients:    make(map[*Client]bool),
			mu:         sync.Mutex{},
		}
		go newHub.run()
		hubs.Store(hubId, newHub)
		return newHub
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}
