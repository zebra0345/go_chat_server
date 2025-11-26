package server

import (
	"context"
	"encoding/json"
	"go-user-server/internal/client"
	"go-user-server/internal/hub"
	"go-user-server/internal/model"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// 시간 설정
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// 데모용으로 모두 허용
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 전체 서버 실행
// hub 생성 , http 서버 Listen, ctx.Done() 혹은 서버 에러시 shutdown
func Run(ctx context.Context, addr string) error {
	h := hub.NewHub()
	go h.Run()

	mux := http.NewServeMux()

	// 정적 파일 서빙
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	// 웹소켓 엔드포인트
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(h, w, r)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	errChan := make(chan error, 1)

	go func() {
		log.Printf("서버 시작 : %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}

	}() // 즉시 실행

	// 컨텍스트 종료 or 서버 에러 대기
	select {
	case <-ctx.Done():
		log.Println("컨텍스트가 종료되었습니다.")
	case err := <-errChan:
		log.Printf("서버 에러가 발생했습니다. ", err)
	}
	h.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("서버가 에러로 인해 종료되었습니다 : ", err)
	}

	log.Println("server exited cleanly")
	return nil
}

// 소켓으로 업그레이드, 허브에 등록, 고루틴 시작하는 함수
func serveWs(h *hub.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	nickName := h.NextNickName()
	c := client.NewClient(conn, nickName)

	h.RegisterClient(c)

	// 첫 메세지 전송
	welcome := model.Message{
		Type:     "welcome",
		Nickname: c.Nickname,
		Message:  "어서오세요" + c.Nickname,
	}

	if b, err := json.Marshal(welcome); err == nil {
		c.Send <- b
	}

	go writePump(c)
	go readPump(h, c)
}

// 텍스트 채팅으로 브로드캐스트
func readPump(h *hub.Hub, c *client.Client) {
	defer func() {
		h.UnregisterClient(c)
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})	

	for {
		msgType, data, err := c.Conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("read error (%s): %v", c.Nickname, err)
			}
			break
		}

		if msgType != websocket.TextMessage {
			continue
		}

		// 클라이언트가 보낸 텍스트를 chat타입으로 브로드캐스트
		chat := model.Message{
			Type:     "chat",
			Nickname: c.Nickname,
			Message:  string(data),
		}
		if b, err := json.Marshal(chat); err == nil {
			h.Broadcast(b)
		}
	}
}

func writePump(c *client.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Send 채널이 닫힌 경우: 종료
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// 주기적으로 ping
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
