package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type InMemoryTokenStorage struct {
	TokenMap map[string]string
	mutex    sync.Mutex
}

func NewInMemoryTokenStorage() *InMemoryTokenStorage {
	return &InMemoryTokenStorage{
		TokenMap: make(map[string]string),
	}
}

type RedisTokenStorage struct {
	client    *redis.Client
	namespace string
}

func NewRedisTokenStorage(client *redis.Client, namespace string) *RedisTokenStorage {
	return &RedisTokenStorage{client: client, namespace: namespace}
}

// Should be safe to use in concurreny
type TokenStorage interface {
	// Store given token for the given email address,
	// returns an error when it somehow fails to store the value.
	// Should not return an error when the value already exists,
	// it should just update in that case.
	StoreToken(email, token string) error

	// Should retrieve the token for the given email address
	// and return an error in any case where it fails to do so.
	RetrieveToken(email string) (string, error)

	// Should remove the token and return an error if it fails to do so.
	// The value not being there should also be considered an error.
	RemoveToken(email string) error
}

// ------------------------------------------------------------------------------

func createKey(namespace, email string) string {
	return fmt.Sprintf("%s:token:%s", namespace, email)
}

const Timeout time.Duration = 24 * time.Hour

func (s *RedisTokenStorage) StoreToken(email, token string) error {
	ctx := context.Background()
	return s.client.Set(ctx, createKey(s.namespace, email), token, Timeout).Err()
}

func (s *RedisTokenStorage) RetrieveToken(email string) (string, error) {
	ctx := context.Background()
	return s.client.Get(ctx, createKey(s.namespace, email)).Result()
}

func (s *RedisTokenStorage) RemoveToken(email string) error {
	ctx := context.Background()
	return s.client.Del(ctx, createKey(s.namespace, email)).Err()
}

// ------------------------------------------------------------------------------

func (s *InMemoryTokenStorage) StoreToken(email, token string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.TokenMap[email] = token
	return nil
}

func (s *InMemoryTokenStorage) RetrieveToken(email string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if token, ok := s.TokenMap[email]; ok {
		return token, nil
	} else {
		return "", fmt.Errorf("failed to find token for %s", email)
	}
}

func (s *InMemoryTokenStorage) RemoveToken(email string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.TokenMap[email]; ok {
		delete(s.TokenMap, email)
		return nil
	} else {
		return fmt.Errorf("failed to remove token for %s, because it wasn't there", email)
	}
}
