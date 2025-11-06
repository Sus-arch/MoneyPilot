package websockets

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHub управляет всеми подключениями
type WebSocketHub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func NewHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[*websocket.Conn]bool),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // разрешаем все источники
}

// HandleConnection — endpoint для подключения фронта
func (h *WebSocketHub) HandleConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("[ws] upgrade error:", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	log.Println("[ws] new client connected")

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			log.Println("[ws] client disconnected")
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// Broadcast — отправка сообщения всем подключённым клиентам
func (h *WebSocketHub) Broadcast(message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			log.Println("[ws] write error:", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
