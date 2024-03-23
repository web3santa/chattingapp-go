package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait = 10 * time.Second

	pingInterval = (pongWait * 9) / 10
)

type ClientList map[*Client]bool

type Client struct {
	//
	connection *websocket.Conn

	manager *Manager

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
	}
}

// func (c *Client) readMessages() {
// 	defer func() {
// 		// cleanup connection
// 		c.manager.removeClient(c)
// 	}()

// 	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
// 		log.Println(err)
// 		return
// 	}

// 	c.connection.SetReadLimit(512)

// 	c.connection.SetPongHandler(c.pongHandler)

// 	for {
// 		_, payload, err := c.connection.ReadMessage()

// 		if err != nil {
// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				log.Printf("error reading message: %v", err)
// 			}
// 			break
// 		}

// 		var request Event

// 		if err := json.Unmarshal(payload, &request); err != nil {
// 			log.Printf("error marchalling evnet : %v", err)
// 			break
// 		}

// 		if err := c.manager.routeEvent(request, c); err != nil {
// 			log.Printf("error handling evnet : %v", err)
// 		}
// 	}
// }

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				// 채널이 닫히면 함수 종료
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println("Error marshaling message:", err)
				continue // 오류 발생 시 반복문 진행
			}

			err = c.connection.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Println("Error sending message:", err)
				return // 오류 발생 시 함수 종료
			}

			log.Println("Message sent")

		case <-ticker.C:
			err := c.connection.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Println("Error sending ping:", err)
				return // 오류 발생 시 함수 종료
			}
			log.Println("Ping sent")
		}
	}
}

func (c *Client) readMessages() {
	defer func() {
		// 클라이언트 제거
		c.manager.removeClient(c)
	}()

	// 연결에 읽기 타임아웃 설정
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println("Error setting read deadline:", err)
		return
	}

	// 최대 읽기 크기 설정
	c.connection.SetReadLimit(512)

	// 핑 메시지 핸들러 설정
	c.connection.SetPongHandler(c.pongHandler)

	for {
		// 메시지 읽기
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			// 오류 발생 시 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			}
			break
		}

		// 읽은 메시지를 이벤트로 언마샬링
		var request Event
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("Error unmarshalling event: %v", err)
			break
		}

		// 이벤트를 관리자로 라우팅
		if err := c.manager.routeEvent(request, c); err != nil {
			log.Printf("Error routing event: %v", err)
		}
	}
}

// func (c *Client) writeMessages() {
// 	defer func() {
// 		c.manager.removeClient(c)
// 	}()

// 	ticker := time.NewTicker(pingInterval)

// 	for {
// 		select {
// 		case message, ok := <-c.egress:
// 			if !ok {
// 				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
// 					log.Println("connection closed: ", err)
// 				}
// 				return
// 			}

// 			data, err := json.Marshal(message)
// 			if err != nil {
// 				log.Println(err)
// 				return
// 			}

// 			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
// 				log.Printf("failed to send message: %v", err)
// 			}
// 			log.Println("message sent")

// 		case <-ticker.C:
// 			log.Println("ping")
// 			// send a ping to the client
// 			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
// 				log.Println("Writemsg err: ", err)
// 				return
// 			}
// 		}

// 	}

// }

func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
