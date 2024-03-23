package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	// 필요한 필드들을 추가할 수 있습니다.
	clients ClientList
	sync.RWMutex

	otps RetentionMap

	handlers map[string]EventHandler
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		handlers: make(map[string]EventHandler),
		otps:     NewRetentionMap(ctx, 5*time.Second),
	}
	m.setupEventHandlers()
	return m

}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
}

func SendMessage(event Event, c *Client) error {
	var chatevent SendMessageEvent

	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return fmt.Errorf("bas payload in request: %v", err)
	}

	var broadMessage NewMessageEvent

	broadMessage.Sent = time.Now()
	broadMessage.Message = chatevent.Message
	broadMessage.From = chatevent.From

	data, err := json.Marshal(broadMessage)
	if err != nil {
		return fmt.Errorf("failed to marchal broadcast message: %v", err)
	}

	println(data)

	outgoingEvent := Event{
		Payload: data,
		// Type:    EventSendMessage,
		Type: EventNewMessage,
	}

	for client := range c.manager.clients {
		client.egress <- outgoingEvent
	}
	return nil
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	// 이벤트 유형에 해당하는 핸들러 찾기
	handler, ok := m.handlers[event.Type]
	if !ok {
		// 처리되지 않은 이벤트 유형에 대한 에러 처리
		return fmt.Errorf("no handler found for event type: %s", event.Type)
	}

	// 핸들러가 nil인지 확인하고 호출
	if handler != nil {
		if err := handler(event, c); err != nil {
			return err
		}
	}

	return nil
}

// func (m *Manager) routeEvent(event Event, c *Client) error {
// 	// check if the event type
// 	if handler, ok := m.handlers[event.Type]; ok {
// 		if err := handler(event, c); err != nil {
// 			return err
// 		}
// 		return nil
// 	} else {
// 		return nil
// 	}
// }

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !m.otps.VerifyOTP(otp) {
		return
	}

	log.Println("new connection")

	// upgrade regular http connection into websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := NewClient(conn, m)

	m.addClient(client)

	// start client processes
	go client.readMessages()
	go client.writeMessages()

}

func (m *Manager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest

	err := decodeJsonRequest(r, &req)
	if err != nil {
		http.Error(w, "Failed to decode JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "percy" && req.Password == "123" {
		type response struct {
			OTP string `json:"otp"`
		}

		otp := m.otps.NewOTP()

		resp := response{
			OTP: otp.KEY,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
}

func decodeJsonRequest(r *http.Request, data interface{}) error {
	return json.NewDecoder(r.Body).Decode(data)
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	// 클라이언트가 이미 목록에 있는지 확인
	if _, ok := m.clients[client]; !ok {
		// 목록에 없으면 클라이언트 추가
		m.clients[client] = true
	}
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	// 클라이언트를 목록에서 제거하고 연결 닫기
	if _, ok := m.clients[client]; ok {
		// 클라이언트가 목록에 있는 경우에만 제거
		delete(m.clients, client)
		client.connection.Close()
	}
}

// func (m *Manager) addClient(client *Client) {
// 	m.Lock()
// 	defer m.Unlock()

// 	if _, ok := m.clients[client]; ok {
// 		m.clients[client] = true
// 	}
// }

// func (m *Manager) removeClient(client *Client) {
// 	m.Lock()
// 	defer m.Unlock()

// 	if _, ok := m.clients[client]; ok {
// 		client.connection.Close()
// 		delete(m.clients, client)
// 	}
// }

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "https://localhost:3000":
		return true
	default:
		return false
	}
}
