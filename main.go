package main

import (
	"context"
	"log"
	"net/http"
)

func setupAPI() {
	ctx := context.Background()
	manager := NewManager(ctx)

	// 정적 파일 제공을 위한 핸들러 등록
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	// 웹 소켓 연결 핸들러 등록
	http.HandleFunc("/ws", manager.ServeWS)
	http.HandleFunc("/login", manager.LoginHandler)
}

func main() {
	setupAPI()
	log.Fatal(http.ListenAndServeTLS(":3000", "server.crt", "server.key", nil))
}
