package client

import "github.com/gorilla/websocket"

type Client struct {
	Nickname string
	Conn *websocket.Conn
	Send chan []byte
}

// NewClient 는 새로운 Client 인스턴스를 생성한다.
// Send 채널에 버퍼를 주어, 약간 느린 클라이언트 때문에 서버가 막히지 않도록 한다.
func NewClient(conn *websocket.Conn, nickname string) *Client {
	return &Client{
		Nickname: nickname,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}
}
