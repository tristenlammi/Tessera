package websocket

import (
	"encoding/json"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler handles WebSocket connections
type Handler struct {
	hub *Hub
	log zerolog.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, log zerolog.Logger) *Handler {
	return &Handler{
		hub: hub,
		log: log,
	}
}

// ClientMessage represents an incoming message from a client
type ClientMessage struct {
	Action   string          `json:"action"`
	FolderID *uuid.UUID      `json:"folder_id,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// HandleConnection handles a new WebSocket connection
func (h *Handler) HandleConnection(c *websocket.Conn) {
	// Get user ID from locals (set by auth middleware)
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		h.log.Error().Msg("No user ID in WebSocket connection")
		c.Close()
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error().Err(err).Msg("Invalid user ID in WebSocket connection")
		c.Close()
		return
	}

	client := &Client{
		ID:      uuid.New().String(),
		UserID:  userID,
		Conn:    c,
		Send:    make(chan []byte, 256),
		Hub:     h.hub,
		Folders: make(map[uuid.UUID]bool),
	}

	h.hub.register <- client

	// Send welcome message
	welcome := &Event{
		Type:      "connected",
		Payload:   map[string]string{"client_id": client.ID},
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
	}
	if data, err := json.Marshal(welcome); err == nil {
		client.Send <- data
	}

	// Start goroutines for reading and writing
	go h.writePump(client)
	h.readPump(client)
}

// readPump pumps messages from the WebSocket connection to the hub
func (h *Handler) readPump(client *Client) {
	defer func() {
		h.hub.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.log.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		h.handleMessage(client, message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (h *Handler) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// Hub closed the channel
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				h.log.Error().Err(err).Msg("WebSocket write error")
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes an incoming message from a client
func (h *Handler) handleMessage(client *Client, message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		h.log.Error().Err(err).Msg("Failed to unmarshal client message")
		return
	}

	switch msg.Action {
	case "subscribe":
		if msg.FolderID != nil {
			h.hub.SubscribeFolder(client, *msg.FolderID)
			h.sendAck(client, "subscribed", map[string]interface{}{
				"folder_id": msg.FolderID.String(),
			})
		} else {
			// Subscribe to root folder (null parent)
			rootID := uuid.Nil
			h.hub.SubscribeFolder(client, rootID)
			h.sendAck(client, "subscribed", map[string]interface{}{
				"folder_id": nil,
			})
		}

	case "unsubscribe":
		if msg.FolderID != nil {
			h.hub.UnsubscribeFolder(client, *msg.FolderID)
			h.sendAck(client, "unsubscribed", map[string]interface{}{
				"folder_id": msg.FolderID.String(),
			})
		} else {
			rootID := uuid.Nil
			h.hub.UnsubscribeFolder(client, rootID)
			h.sendAck(client, "unsubscribed", map[string]interface{}{
				"folder_id": nil,
			})
		}

	case "ping":
		h.sendAck(client, "pong", nil)

	default:
		h.log.Warn().Str("action", msg.Action).Msg("Unknown WebSocket action")
	}
}

func (h *Handler) sendAck(client *Client, action string, data map[string]interface{}) {
	response := map[string]interface{}{
		"type":      "ack",
		"action":    action,
		"timestamp": time.Now().UnixMilli(),
	}
	if data != nil {
		response["data"] = data
	}

	if jsonData, err := json.Marshal(response); err == nil {
		select {
		case client.Send <- jsonData:
		default:
			// Client buffer full, skip
		}
	}
}
