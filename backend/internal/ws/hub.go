package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/trello-clone/backend/internal/auth"
	"github.com/trello-clone/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type client struct {
	topic string
	conn  *websocket.Conn
	send  chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
	redis   *redis.Client
	tokens  *auth.TokenManager
}

func NewHub(rdb *redis.Client, tm *auth.TokenManager) *Hub {
	h := &Hub{
		clients: make(map[string]map[*client]struct{}),
		redis:   rdb,
		tokens:  tm,
	}
	if rdb != nil {
		go h.subscribeRedis()
	}
	return h
}

func (h *Hub) subscribeRedis() {
	ctx := context.Background()
	pubsub := h.redis.PSubscribe(ctx, "board:*", "user:*")
	defer pubsub.Close()
	for msg := range pubsub.Channel() {
		h.localBroadcast(msg.Channel, []byte(msg.Payload))
	}
}

func boardTopic(boardID uuid.UUID) string { return "board:" + boardID.String() }
func userTopic(userID uuid.UUID) string   { return "user:" + userID.String() }

// Broadcast sends an event to everyone subscribed to a board.
func (h *Hub) Broadcast(boardID uuid.UUID, event models.WSEvent) {
	event.BoardID = boardID
	h.publish(boardTopic(boardID), event)
}

// BroadcastUser sends an event to a single user across all their connections.
func (h *Hub) BroadcastUser(userID uuid.UUID, event models.WSEvent) {
	h.publish(userTopic(userID), event)
}

func (h *Hub) publish(topic string, event models.WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	// When Redis is enabled, deliver only through the subscriber to avoid
	// double-delivering to locally connected clients.
	if h.redis != nil {
		if err := h.redis.Publish(context.Background(), topic, data).Err(); err != nil {
			log.Printf("redis publish error: %v", err)
		}
		return
	}
	h.localBroadcast(topic, data)
}

func (h *Hub) localBroadcast(topic string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[topic] {
		select {
		case c.send <- data:
		default:
		}
	}
}

// HandleWS upgrades a board-scoped websocket connection.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	boardID, err := uuid.Parse(r.URL.Query().Get("board_id"))
	if err != nil {
		http.Error(w, "invalid board_id", http.StatusBadRequest)
		return
	}
	h.serve(w, r, boardTopic(boardID))
}

// HandleUserWS upgrades a per-user websocket connection used for notifications.
func (h *Hub) HandleUserWS(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	h.serve(w, r, userTopic(claims.UserID))
}

func (h *Hub) authenticate(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return nil, false
	}
	claims, err := h.tokens.ParseAccessToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return nil, false
	}
	return claims, true
}

func (h *Hub) serve(w http.ResponseWriter, r *http.Request, topic string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{topic: topic, conn: conn, send: make(chan []byte, 64)}
	h.register(c)
	defer func() {
		h.unregister(c)
		conn.Close()
	}()
	go c.writePump()
	c.readPump()
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.topic] == nil {
		h.clients[c.topic] = make(map[*client]struct{})
	}
	h.clients[c.topic][c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.topic]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.topic)
		}
	}
	close(c.send)
}

func (c *client) readPump() {
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *client) writePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}
