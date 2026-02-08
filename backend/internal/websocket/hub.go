package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// EventType represents the type of WebSocket event
type EventType string

const (
	EventFileCreated    EventType = "file:created"
	EventFileUpdated    EventType = "file:updated"
	EventFileDeleted    EventType = "file:deleted"
	EventFileMoved      EventType = "file:moved"
	EventFileRestored   EventType = "file:restored"
	EventUploadStarted  EventType = "upload:started"
	EventUploadProgress EventType = "upload:progress"
	EventUploadComplete EventType = "upload:complete"
	EventShareCreated   EventType = "share:created"
	EventShareRevoked   EventType = "share:revoked"
	EventStorageUpdated EventType = "storage:updated"
)

// Event represents a WebSocket event
type Event struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	FolderID  *uuid.UUID  `json:"folder_id,omitempty"`
	UserID    uuid.UUID   `json:"user_id"`
	Timestamp int64       `json:"timestamp"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID      string
	UserID  uuid.UUID
	Conn    *websocket.Conn
	Send    chan []byte
	Hub     *Hub
	Folders map[uuid.UUID]bool // Subscribed folder IDs
	mu      sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	clients map[uuid.UUID]map[*Client]bool

	// Folder subscriptions: folderID -> clients
	folders map[uuid.UUID]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast to specific user
	userBroadcast chan *userMessage

	// Broadcast to folder subscribers
	folderBroadcast chan *folderMessage

	// Broadcast to all users
	broadcast chan []byte

	mu  sync.RWMutex
	log zerolog.Logger
}

type userMessage struct {
	userID  uuid.UUID
	message []byte
}

type folderMessage struct {
	folderID uuid.UUID
	message  []byte
	exclude  *Client // Optional client to exclude
}

// NewHub creates a new Hub instance
func NewHub(log zerolog.Logger) *Hub {
	return &Hub{
		clients:         make(map[uuid.UUID]map[*Client]bool),
		folders:         make(map[uuid.UUID]map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		userBroadcast:   make(chan *userMessage, 256),
		folderBroadcast: make(chan *folderMessage, 256),
		broadcast:       make(chan []byte, 256),
		log:             log,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.userBroadcast:
			h.sendToUser(msg.userID, msg.message)

		case msg := <-h.folderBroadcast:
			h.sendToFolder(msg.folderID, msg.message, msg.exclude)

		case message := <-h.broadcast:
			h.sendToAll(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true

	h.log.Debug().
		Str("client_id", client.ID).
		Str("user_id", client.UserID.String()).
		Msg("Client registered")
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.UserID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.Send)

			// Clean up folder subscriptions
			client.mu.RLock()
			for folderID := range client.Folders {
				if h.folders[folderID] != nil {
					delete(h.folders[folderID], client)
					if len(h.folders[folderID]) == 0 {
						delete(h.folders, folderID)
					}
				}
			}
			client.mu.RUnlock()

			if len(clients) == 0 {
				delete(h.clients, client.UserID)
			}

			h.log.Debug().
				Str("client_id", client.ID).
				Str("user_id", client.UserID.String()).
				Msg("Client unregistered")
		}
	}
}

func (h *Hub) sendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	clients := h.clients[userID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.Send <- message:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) sendToFolder(folderID uuid.UUID, message []byte, exclude *Client) {
	h.mu.RLock()
	clients := h.folders[folderID]
	h.mu.RUnlock()

	for client := range clients {
		if client == exclude {
			continue
		}
		select {
		case client.Send <- message:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) sendToAll(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				h.unregister <- client
			}
		}
	}
}

// SubscribeFolder adds a client to a folder's subscription list
func (h *Hub) SubscribeFolder(client *Client, folderID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.folders[folderID] == nil {
		h.folders[folderID] = make(map[*Client]bool)
	}
	h.folders[folderID][client] = true

	client.mu.Lock()
	client.Folders[folderID] = true
	client.mu.Unlock()

	h.log.Debug().
		Str("client_id", client.ID).
		Str("folder_id", folderID.String()).
		Msg("Client subscribed to folder")
}

// UnsubscribeFolder removes a client from a folder's subscription list
func (h *Hub) UnsubscribeFolder(client *Client, folderID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.folders[folderID] != nil {
		delete(h.folders[folderID], client)
		if len(h.folders[folderID]) == 0 {
			delete(h.folders, folderID)
		}
	}

	client.mu.Lock()
	delete(client.Folders, folderID)
	client.mu.Unlock()
}

// BroadcastToUser sends an event to all connections of a specific user
func (h *Hub) BroadcastToUser(userID uuid.UUID, event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to marshal event")
		return
	}

	h.userBroadcast <- &userMessage{
		userID:  userID,
		message: data,
	}
}

// BroadcastToFolder sends an event to all subscribers of a folder
func (h *Hub) BroadcastToFolder(folderID uuid.UUID, event *Event, exclude *Client) {
	data, err := json.Marshal(event)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to marshal event")
		return
	}

	h.folderBroadcast <- &folderMessage{
		folderID: folderID,
		message:  data,
		exclude:  exclude,
	}
}

// BroadcastToAll sends an event to all connected clients
func (h *Hub) BroadcastToAll(event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to marshal event")
		return
	}

	h.broadcast <- data
}

// GetOnlineUsers returns a list of currently online user IDs
func (h *Hub) GetOnlineUsers() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]uuid.UUID, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// IsUserOnline checks if a user has any active connections
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

// GetConnectionCount returns the total number of active connections
func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}
