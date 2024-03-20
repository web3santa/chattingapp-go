package main

import (
	"log"
	"net/http"
)

func setupAPI() {
	manager := NewManager()

	// 정적 파일 제공을 위한 핸들러 등록
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	// 웹 소켓 연결 핸들러 등록
	http.HandleFunc("/ws", manager.ServeWS)
}

func main() {
	setupAPI()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
