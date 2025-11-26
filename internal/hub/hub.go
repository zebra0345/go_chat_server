package hub

import (
	"encoding/json"
	"go-user-server/internal/client"
	"go-user-server/internal/model"
	"log"
	"strconv"
	"sync"
)

// Hub 는 모든 클라이언트와 브로드캐스트를 관리하는 중앙 허브이다.
type Hub struct {
	register   chan *client.Client
	unregister chan *client.Client
	broadcast  chan []byte

	clients map[*client.Client]struct{}

	mu       sync.Mutex
	nextUser int

	done chan struct{}
}

// Hub 초기화
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *client.Client),
		unregister: make(chan *client.Client),
		broadcast:  make(chan []byte),
		clients:    make(map[*client.Client]struct{}),
		nextUser:   1,
		done:       make(chan struct{}),
	}
}

// 허브 메인 이벤트 루프
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = struct{}{}
			log.Printf("채팅방에 입장하셨습니다 : %s", c.Nickname)

			// 입장 시스템 메시지 브로드캐스트
			msg := model.Message{
				Type:     "system",
				Nickname: c.Nickname,
				Message:  c.Nickname + " 입장",
			}
			if b, err := json.Marshal(msg); err == nil {
				h.broadcastToAll(b)
			}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
				_ = c.Conn.Close()
				log.Printf("로그아웃 : %s", c.Nickname)

				// 퇴장 시스템 메시지 브로드캐스트
				msg := model.Message{
					Type:     "system",
					Nickname: c.Nickname,
					Message:  c.Nickname + " 퇴장",
				}
				if b, err := json.Marshal(msg); err == nil {
					h.broadcastToAll(b)
				}
			}

		case message := <-h.broadcast:
			// 일반 채팅 메시지 브로드캐스트
			h.broadcastToAll(message)

		case <-h.done:
			// 모든 사용자 정리
			for c := range h.clients {
				close(c.Send)
				_ = c.Conn.Close()
				delete(h.clients, c)
			}
			log.Println("허브 종료")
			return
		}
	}
}

// 모든 클라이언트에게 메시지를 뿌리는 헬퍼
func (h *Hub) broadcastToAll(message []byte) {
	for c := range h.clients {
		select {
		case c.Send <- message:
		default:
			// 너무 느린 클라이언트는 정리
			close(c.Send)
			_ = c.Conn.Close()
			delete(h.clients, c)
		}
	}
}

// 닉네임 생성: userN 형식 (뮤텍스로 보호)
func (h *Hub) NextNickName() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := h.nextUser
	h.nextUser++
	return "user" + strconv.Itoa(n)
}

// Stop 은 허브를 종료시키는 신호를 보낸다.
func (h *Hub) Stop() {
	close(h.done)
}

// RegisterClient 은 외부에서 클라이언트를 등록하는 public API.
func (h *Hub) RegisterClient(c *client.Client) {
	h.register <- c
}

// UnregisterClient 은 외부에서 클라이언트를 해제하는 public API.
func (h *Hub) UnregisterClient(c *client.Client) {
	h.unregister <- c
}

// Broadcast 는 외부에서 메시지를 브로드캐스트 채널로 밀어 넣는 API.
func (h *Hub) Broadcast(b []byte) {
	h.broadcast <- b
}
