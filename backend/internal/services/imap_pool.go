package services

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rs/zerolog/log"
	"github.com/tessera/tessera/internal/models"
)

// IMAPPool manages reusable IMAP connections per account
type IMAPPool struct {
	mu    sync.Mutex
	pools map[string]*accountPool // key: accountID
}

type accountPool struct {
	mu          sync.Mutex
	connections []*pooledConnection
	account     *models.EmailAccount
	maxSize     int
	maxAge      time.Duration
}

type pooledConnection struct {
	client    *imapclient.Client
	createdAt time.Time
	inUse     bool
}

func NewIMAPPool() *IMAPPool {
	return &IMAPPool{
		pools: make(map[string]*accountPool),
	}
}

// Get retrieves or creates an IMAP connection for the given account
func (p *IMAPPool) Get(account *models.EmailAccount) (*imapclient.Client, error) {
	p.mu.Lock()
	pool, ok := p.pools[account.ID]
	if !ok {
		pool = &accountPool{
			account: account,
			maxSize: 3,
			maxAge:  5 * time.Minute,
		}
		p.pools[account.ID] = pool
	}
	p.mu.Unlock()

	return pool.get()
}

// Return returns a connection to the pool for reuse
func (p *IMAPPool) Return(accountID string, client *imapclient.Client) {
	p.mu.Lock()
	pool, ok := p.pools[accountID]
	p.mu.Unlock()

	if !ok || client == nil {
		if client != nil {
			client.Close()
		}
		return
	}

	pool.put(client)
}

// Close closes a connection without returning it to the pool
func (p *IMAPPool) Close(client *imapclient.Client) {
	if client != nil {
		client.Close()
	}
}

// CloseAll closes all pooled connections
func (p *IMAPPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pool := range p.pools {
		pool.closeAll()
	}
	p.pools = make(map[string]*accountPool)
}

// CloseAccount closes all pooled connections for a specific account
func (p *IMAPPool) CloseAccount(accountID string) {
	p.mu.Lock()
	pool, ok := p.pools[accountID]
	if ok {
		delete(p.pools, accountID)
	}
	p.mu.Unlock()

	if ok {
		pool.closeAll()
	}
}

func (ap *accountPool) get() (*imapclient.Client, error) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Try to find an available connection
	now := time.Now()
	for i := len(ap.connections) - 1; i >= 0; i-- {
		conn := ap.connections[i]

		// Remove expired connections
		if now.Sub(conn.createdAt) > ap.maxAge {
			conn.client.Close()
			ap.connections = append(ap.connections[:i], ap.connections[i+1:]...)
			continue
		}

		if !conn.inUse {
			// Test if connection is still alive with a simple NOOP
			if err := conn.client.Noop().Wait(); err != nil {
				// Connection is dead, remove it
				conn.client.Close()
				ap.connections = append(ap.connections[:i], ap.connections[i+1:]...)
				continue
			}
			conn.inUse = true
			return conn.client, nil
		}
	}

	// No available connection, create a new one
	client, err := connectIMAP(ap.account)
	if err != nil {
		return nil, err
	}

	if len(ap.connections) < ap.maxSize {
		ap.connections = append(ap.connections, &pooledConnection{
			client:    client,
			createdAt: now,
			inUse:     true,
		})
	}

	return client, nil
}

func (ap *accountPool) put(client *imapclient.Client) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Find this client in our pool and mark as not in use
	for _, conn := range ap.connections {
		if conn.client == client {
			conn.inUse = false
			return
		}
	}

	// Client wasn't in pool (pool was full when created) - close it
	client.Close()
}

func (ap *accountPool) closeAll() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	for _, conn := range ap.connections {
		conn.client.Close()
	}
	ap.connections = nil
}

// connectIMAP creates a new IMAP connection (standalone function for pool use)
func connectIMAP(account *models.EmailAccount) (*imapclient.Client, error) {
	addr := fmt.Sprintf("%s:%d", account.IMAPHost, account.IMAPPort)

	var client *imapclient.Client
	var err error

	if account.IMAPUseTLS {
		client, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: account.IMAPHost},
		})
	} else {
		client, err = imapclient.DialInsecure(addr, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := client.Login(account.IMAPUsername, account.IMAPPassword).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("login failed: %w", err)
	}

	return client, nil
}

// withRetry executes a function with exponential backoff retry
func withRetry(maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt < maxAttempts-1 {
			delay := time.Duration(1<<uint(attempt)) * 500 * time.Millisecond
			if delay > 10*time.Second {
				delay = 10 * time.Second
			}
			log.Warn().Err(lastErr).Int("attempt", attempt+1).Dur("retry_in", delay).Msg("IMAP operation failed, retrying")
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
