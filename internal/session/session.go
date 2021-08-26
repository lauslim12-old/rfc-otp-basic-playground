package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-redis/redis/v8"
)

// Everything on this codebase will use the background context.
var ctx = context.Background()

// Service represents the dependency of this package.
type Service struct {
	redis             *redis.Client
	sessionExpiration time.Duration
}

// NewService creates a new service to be used to perform operations with the Redis.
func New(redis *redis.Client, sessionExpiration time.Duration) *Service {
	return &Service{
		redis:             redis,
		sessionExpiration: sessionExpiration,
	}
}

// GenerateSessionID is used to generate URL-safe, base64 encoded, secure generated random string.
// 32 bytes should be enough for cryptographically safe generation (256 bits).
// Will return an error if the system's secure random number generator fails to perform properly.
func GenerateSessionID(numberOfBytes int) (string, error) {
	b := make([]byte, numberOfBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// Set is to set a new session ID that is connected with the user ID.
// Redis's 'SET' can't fail.
func (s *Service) Set(sessionID, userID string) error {
	_, err := s.redis.Set(ctx, sessionID, userID, s.sessionExpiration).Result()
	if err != nil {
		return err
	}

	return nil
}

// Get is to get the user ID that is associated with the session ID.
func (s *Service) Get(sessionID string) (string, error) {
	res, err := s.redis.Get(ctx, sessionID).Result()
	if err != nil && err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return res, nil
}
