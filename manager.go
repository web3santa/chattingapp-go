package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	// 필요한 필드들을 추가할 수 있습니다.
	clients ClientList
	sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		clients: make(ClientList),
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")

	// upgrade regular http connection into websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := NewClient(conn, m)

	m.addClient(client)
	conn.Close()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		m.clients[client] = true
	}
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		m.clients[client] = true
	}
}
