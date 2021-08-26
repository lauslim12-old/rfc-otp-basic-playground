package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

// KeysAndUsers represents an object of a user and their session ID.
type KeyAndUser struct {
	SessionID string `json:"sessionId"`
	UserID    string `json:"userId"`
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
	redisKey := fmt.Sprintf("sess:%s", sessionID)
	_, err := s.redis.Set(ctx, redisKey, userID, s.sessionExpiration).Result()
	if err != nil {
		return err
	}

	return nil
}

// Get is to get the user ID that is associated with the session ID.
func (s *Service) Get(sessionID string) (string, error) {
	redisKey := fmt.Sprintf("sess:%s", sessionID)
	res, err := s.redis.Get(ctx, redisKey).Result()
	if err != nil && err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return res, nil
}

// All is to get all of the currently available sessions.
func (s *Service) All() ([]KeyAndUser, error) {
	var keysCollection []string
	var keysAndUsers []KeyAndUser

	for {
		// Iteratively get all the keys.
		keys, cursor, err := s.redis.Scan(ctx, 0, "sess:*", 10).Result()
		if err != nil && err == redis.Nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		// Append to this variable every time we get a new result.
		keysCollection = append(keysCollection, keys...)
		if cursor == 0 {
			break
		}
	}

	for i := 0; i < len(keysCollection); i += 1 {
		// Get all users and append them to an object, with their session data.
		user, err := s.redis.Get(ctx, keysCollection[i]).Result()
		if err != nil {
			return nil, err
		}

		keysAndUsers = append(keysAndUsers, KeyAndUser{keysCollection[i], user})
	}

	return keysAndUsers, nil
}
