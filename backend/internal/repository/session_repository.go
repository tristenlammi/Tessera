package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/tessera/tessera/internal/models"
)

// SessionRepository handles session storage in Redis
type SessionRepository struct {
	rdb *redis.Client
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(rdb *redis.Client) *SessionRepository {
	return &SessionRepository{rdb: rdb}
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func userSessionsKey(userID uuid.UUID) string {
	return fmt.Sprintf("user_sessions:%s", userID.String())
}

func refreshTokenKey(token string) string {
	return fmt.Sprintf("refresh_token:%s", token)
}

func pendingAuthKey(token string) string {
	return fmt.Sprintf("pending_auth:%s", token)
}

func wsTicketKey(ticket string) string {
	return fmt.Sprintf("ws_ticket:%s", ticket)
}

// WebSocketTicket represents a short-lived ticket for WebSocket authentication
type WebSocketTicket struct {
	UserID uuid.UUID `json:"user_id"`
}

// CreateWSTicket creates a short-lived ticket for WebSocket authentication (30 second TTL)
func (r *SessionRepository) CreateWSTicket(ctx context.Context, ticket string, userID uuid.UUID) error {
	data, err := json.Marshal(&WebSocketTicket{UserID: userID})
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, wsTicketKey(ticket), data, 30*time.Second).Err()
}

// GetWSTicket retrieves and deletes a WebSocket ticket atomically (one-time use)
func (r *SessionRepository) GetWSTicket(ctx context.Context, ticket string) (*WebSocketTicket, error) {
	key := wsTicketKey(ticket)
	// Use GetDel for atomic get-and-delete (Redis 6.2+)
	// This prevents TOCTOU race conditions where the same ticket could be used twice
	data, err := r.rdb.GetDel(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	wsTicket := &WebSocketTicket{}
	if err := json.Unmarshal(data, wsTicket); err != nil {
		return nil, err
	}
	return wsTicket, nil
}

// PendingAuth represents a partial authentication awaiting 2FA
type PendingAuth struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// CreatePendingAuth stores a pending auth token for 2FA flow (5 minute TTL)
func (r *SessionRepository) CreatePendingAuth(ctx context.Context, token string, userID uuid.UUID, email string) error {
	data, err := json.Marshal(&PendingAuth{UserID: userID, Email: email})
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, pendingAuthKey(token), data, 5*time.Minute).Err()
}

// GetPendingAuth retrieves and deletes a pending auth token atomically
func (r *SessionRepository) GetPendingAuth(ctx context.Context, token string) (*PendingAuth, error) {
	key := pendingAuthKey(token)
	// Use GetDel for atomic get-and-delete (Redis 6.2+)
	// This prevents TOCTOU race conditions where the same token could be used twice
	data, err := r.rdb.GetDel(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	pending := &PendingAuth{}
	if err := json.Unmarshal(data, pending); err != nil {
		return nil, err
	}
	return pending, nil
}

// Create stores a new session
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)

	pipe := r.rdb.Pipeline()

	// Store session
	pipe.Set(ctx, sessionKey(session.ID), data, ttl)

	// Add to user's sessions set
	pipe.SAdd(ctx, userSessionsKey(session.UserID), session.ID)
	pipe.Expire(ctx, userSessionsKey(session.UserID), ttl)

	// Store refresh token mapping
	pipe.Set(ctx, refreshTokenKey(session.RefreshToken), session.ID, ttl)

	_, err = pipe.Exec(ctx)
	return err
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	data, err := r.rdb.Get(ctx, sessionKey(sessionID)).Bytes()
	if err != nil {
		return nil, err
	}

	session := &models.Session{}
	if err := json.Unmarshal(data, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetByRefreshToken retrieves a session by its refresh token
func (r *SessionRepository) GetByRefreshToken(ctx context.Context, token string) (*models.Session, error) {
	sessionID, err := r.rdb.Get(ctx, refreshTokenKey(token)).Result()
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, sessionID)
}

// Delete removes a session
func (r *SessionRepository) Delete(ctx context.Context, session *models.Session) error {
	pipe := r.rdb.Pipeline()

	pipe.Del(ctx, sessionKey(session.ID))
	pipe.SRem(ctx, userSessionsKey(session.UserID), session.ID)
	pipe.Del(ctx, refreshTokenKey(session.RefreshToken))

	_, err := pipe.Exec(ctx)
	return err
}

// DeleteAllForUser removes all sessions for a user
func (r *SessionRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	// Get all session IDs for user
	sessionIDs, err := r.rdb.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return err
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := r.rdb.Pipeline()

	for _, sid := range sessionIDs {
		// Get session to find refresh token
		data, err := r.rdb.Get(ctx, sessionKey(sid)).Bytes()
		if err == nil {
			session := &models.Session{}
			if json.Unmarshal(data, session) == nil {
				pipe.Del(ctx, refreshTokenKey(session.RefreshToken))
			}
		}
		pipe.Del(ctx, sessionKey(sid))
	}

	pipe.Del(ctx, userSessionsKey(userID))

	_, err = pipe.Exec(ctx)
	return err
}

// GetUserSessions retrieves all active sessions for a user
func (r *SessionRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	sessionIDs, err := r.rdb.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	sessions := make([]*models.Session, 0, len(sessionIDs))

	for _, sid := range sessionIDs {
		session, err := r.GetByID(ctx, sid)
		if err == nil {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}
