package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lovenest/backend/internal/handlers"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev/mobile testing
	},
}

type WSEvent struct {
	Type     string      `json:"type"`
	SenderID int64       `json:"sender_id"`
	Payload  interface{} `json:"payload"`
}

type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	UserID   int64
	CoupleID int64
}

type Hub struct {
	// Registered clients mapped by coupleID -> map of *Client
	couples    map[int64]map[*Client]bool
	broadcast  chan *WSEvent
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

var GlobalHub *Hub

func NewHub() *Hub {
	h := &Hub{
		couples:    make(map[int64]map[*Client]bool),
		broadcast:  make(chan *WSEvent, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	// Connect hooks with handlers package
	handlers.BroadcastSceneEvent = func(coupleID int64, senderID int64, eventType string, payload interface{}) {
		h.BroadcastToCouple(coupleID, &WSEvent{
			Type:     eventType,
			SenderID: senderID,
			Payload:  payload,
		})
	}

	handlers.BroadcastChatMessage = func(coupleID int64, senderID int64, message models.Message) {
		h.BroadcastToCouple(coupleID, &WSEvent{
			Type:     "chat:message",
			SenderID: senderID,
			Payload:  message,
		})
	}

	return h
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.couples[client.CoupleID] == nil {
				h.couples[client.CoupleID] = make(map[*Client]bool)
			}
			h.couples[client.CoupleID][client] = true
			h.mu.Unlock()

			log.Printf("[WS] User %d connected to couple %d", client.UserID, client.CoupleID)
			// Notify partner of online presence
			h.BroadcastToCouple(client.CoupleID, &WSEvent{
				Type:     "partner:presence",
				SenderID: client.UserID,
				Payload:  map[string]interface{}{"status": "online", "user_id": client.UserID},
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.couples[client.CoupleID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.couples, client.CoupleID)
					}
				}
			}
			h.mu.Unlock()

			log.Printf("[WS] User %d disconnected from couple %d", client.UserID, client.CoupleID)
			// Notify partner of offline presence
			h.BroadcastToCouple(client.CoupleID, &WSEvent{
				Type:     "partner:presence",
				SenderID: client.UserID,
				Payload:  map[string]interface{}{"status": "offline", "user_id": client.UserID},
			})
		}
	}
}

func (h *Hub) BroadcastToCouple(coupleID int64, event *WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.couples[coupleID]
	if !ok || len(clients) == 0 {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	for client := range clients {
		// Do not echo back to sender unless it's presence or chat
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(clients, client)
		}
	}
}

func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Extract user & couple from token
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	var claims *middleware.JWTClaims

	// Parse with standard jwt
	parsedToken, err := middleware.AuthMiddlewareTokenParse(tokenStr)
	if err != nil || parsedToken == nil {
		// Fallback query param for quick dev testing: ?user_id=1&couple_id=1
		devUID := r.URL.Query().Get("user_id")
		devCID := r.URL.Query().Get("couple_id")
		if devUID != "" {
			uid, _ := strconv.ParseInt(devUID, 10, 64)
			cid, _ := strconv.ParseInt(devCID, 10, 64)
			claims.UserID = uid
			claims.CoupleID = &cid
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	} else {
		claims = parsedToken
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	cid := int64(0)
	if claims.CoupleID != nil {
		cid = *claims.CoupleID
	}

	client := &Client{
		Hub:      hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		UserID:   claims.UserID,
		CoupleID: cid,
	}

	client.Hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(65536)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var event WSEvent
		if err := json.Unmarshal(message, &event); err == nil {
			event.SenderID = c.UserID
			// Handle client sent events (e.g. partner:walk, partner:action)
			c.Hub.BroadcastToCouple(c.CoupleID, &event)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
