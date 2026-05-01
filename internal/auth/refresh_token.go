package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RefreshTokenEntry represents a stored refresh token.
type RefreshTokenEntry struct {
	TokenID    string    `json:"token_id"`
	UserID     string    `json:"user_id"`
	Role       string    `json:"role"`
	DeviceInfo string    `json:"device_info,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// RefreshTokenStore provides storage for refresh tokens.
type RefreshTokenStore interface {
	// Store saves a refresh token entry.
	Store(entry RefreshTokenEntry) error
	// Retrieve fetches a refresh token entry by token ID.
	Retrieve(tokenID string) (*RefreshTokenEntry, error)
	// Revoke removes a refresh token (used during logout or rotation).
	Revoke(tokenID string) error
	// RevokeAllForUser removes all refresh tokens for a user (used for security).
	RevokeAllForUser(userID string) error
	// Count returns the number of active refresh tokens for a user.
	Count(userID string) (int, error)
}

// GenerateRefreshTokenID generates a cryptographically secure random token ID.
func GenerateRefreshTokenID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// --- Redis implementation ---

// RedisRefreshTokenStore implements RefreshTokenStore using Redis.
type RedisRefreshTokenStore struct {
	client *redis.Client
}

// NewRedisRefreshTokenStore creates a new Redis-backed refresh token store.
func NewRedisRefreshTokenStore(client *redis.Client) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{client: client}
}

func (s *RedisRefreshTokenStore) Store(entry RefreshTokenEntry) error {
	ctx := context.Background()
	ttl := time.Until(entry.ExpiresAt)
	if ttl <= 0 {
		return nil
	}

	// Store the token entry
	key := fmt.Sprintf("refresh:%s", entry.TokenID)
	data := fmt.Sprintf("%s|%s|%s|%s|%d|%d",
		entry.UserID, entry.Role, entry.DeviceInfo, entry.IPAddress,
		entry.CreatedAt.Unix(), entry.ExpiresAt.Unix())

	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, data, ttl)
	// Add to user's token set for RevvokeAllForUser
	pipe.SAdd(ctx, fmt.Sprintf("user_refresh:%s", entry.UserID), entry.TokenID)
	pipe.Expire(ctx, fmt.Sprintf("user_refresh:%s", entry.UserID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisRefreshTokenStore) Retrieve(tokenID string) (*RefreshTokenEntry, error) {
	ctx := context.Background()
	val, err := s.client.Get(ctx, fmt.Sprintf("refresh:%s", tokenID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseRefreshEntry(tokenID, val)
}

func (s *RedisRefreshTokenStore) Revoke(tokenID string) error {
	ctx := context.Background()
	// Get entry to find userID for set cleanup
	val, err := s.client.Get(ctx, fmt.Sprintf("refresh:%s", tokenID)).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	entry, err := parseRefreshEntry(tokenID, val)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("refresh:%s", tokenID))
	pipe.SRem(ctx, fmt.Sprintf("user_refresh:%s", entry.UserID), tokenID)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisRefreshTokenStore) RevokeAllForUser(userID string) error {
	ctx := context.Background()
	setKey := fmt.Sprintf("user_refresh:%s", userID)
	tokenIDs, err := s.client.SMembers(ctx, setKey).Result()
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	for _, tid := range tokenIDs {
		pipe.Del(ctx, fmt.Sprintf("refresh:%s", tid))
	}
	pipe.Del(ctx, setKey)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisRefreshTokenStore) Count(userID string) (int, error) {
	ctx := context.Background()
	n, err := s.client.SCard(ctx, fmt.Sprintf("user_refresh:%s", userID)).Result()
	return int(n), err
}

// --- Memory implementation ---

// MemoryRefreshTokenStore implements RefreshTokenStore using in-memory storage.
type MemoryRefreshTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*RefreshTokenEntry // tokenID -> entry
	users  map[string]map[string]struct{} // userID -> set of tokenIDs
}

// NewMemoryRefreshTokenStore creates a new in-memory refresh token store.
func NewMemoryRefreshTokenStore() *MemoryRefreshTokenStore {
	return &MemoryRefreshTokenStore{
		tokens: make(map[string]*RefreshTokenEntry),
		users:  make(map[string]map[string]struct{}),
	}
}

func (s *MemoryRefreshTokenStore) Store(entry RefreshTokenEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[entry.TokenID] = &entry
	if s.users[entry.UserID] == nil {
		s.users[entry.UserID] = make(map[string]struct{})
	}
	s.users[entry.UserID][entry.TokenID] = struct{}{}
	return nil
}

func (s *MemoryRefreshTokenStore) Retrieve(tokenID string) (*RefreshTokenEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.tokens[tokenID]
	if !ok {
		return nil, nil
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, nil
	}
	return entry, nil
}

func (s *MemoryRefreshTokenStore) Revoke(tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.tokens[tokenID]
	if !ok {
		return nil
	}
	delete(s.tokens, tokenID)
	if userTokens, ok := s.users[entry.UserID]; ok {
		delete(userTokens, tokenID)
	}
	return nil
}

func (s *MemoryRefreshTokenStore) RevokeAllForUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if userTokens, ok := s.users[userID]; ok {
		for tid := range userTokens {
			delete(s.tokens, tid)
		}
		delete(s.users, userID)
	}
	return nil
}

func (s *MemoryRefreshTokenStore) Count(userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users[userID]), nil
}

// StartCleanup starts a background goroutine that removes expired tokens.
func (s *MemoryRefreshTokenStore) StartCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanup()
			}
		}
	}()
}

func (s *MemoryRefreshTokenStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0
	for tid, entry := range s.tokens {
		if now.After(entry.ExpiresAt) {
			delete(s.tokens, tid)
			if userTokens, ok := s.users[entry.UserID]; ok {
				delete(userTokens, tid)
			}
			count++
		}
	}
	if count > 0 {
		slog.Debug("cleaned up expired refresh tokens", "count", count)
	}
}

// parseRefreshEntry parses a stored refresh token string back into an entry.
func parseRefreshEntry(tokenID, data string) (*RefreshTokenEntry, error) {
	parts := splitN(data, '|', 6)
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid refresh token data")
	}
	createdAt, _ := time.Parse(time.RFC3339, parts[4])
	expiresAt, _ := time.Parse(time.RFC3339, parts[5])
	if createdAt.IsZero() {
		createdAt = time.Unix(parseUnix(parts[4]), 0)
	}
	if expiresAt.IsZero() {
		expiresAt = time.Unix(parseUnix(parts[5]), 0)
	}
	return &RefreshTokenEntry{
		TokenID:    tokenID,
		UserID:     parts[0],
		Role:       parts[1],
		DeviceInfo: parts[2],
		IPAddress:  parts[3],
		CreatedAt:  createdAt,
		ExpiresAt:  expiresAt,
	}, nil
}

func splitN(s string, sep rune, n int) []string {
	return splitString(s, sep, n)
}

func splitString(s string, sep rune, n int) []string {
	var parts []string
	start := 0
	for i, r := range s {
		if r == sep && len(parts) < n-1 {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func parseUnix(s string) int64 {
	n := int64(0)
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			break
		}
	}
	return n
}
