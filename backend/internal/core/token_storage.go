package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type InMemoryTokenStorage struct {
	TokenMap     map[string]string
	LinkTokenMap map[string]string
	mutex        sync.Mutex
}

func NewInMemoryTokenStorage() *InMemoryTokenStorage {
	return &InMemoryTokenStorage{
		TokenMap:     make(map[string]string),
		LinkTokenMap: make(map[string]string),
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

	// StoreLinkToken stores a reverse mapping from an opaque link token to an
	// email address. This lets the verification link carry only the opaque
	// token while the email is looked up server-side, keeping the email out of
	// the URL (and thus out of browser history, server logs and Referer).
	StoreLinkToken(linkToken, email string) error

	// RetrieveEmailByLinkToken returns the email address associated with the
	// given link token, or an error if it is unknown or expired.
	RetrieveEmailByLinkToken(linkToken string) (string, error)

	// RemoveLinkToken removes the given link token mapping. The value not being
	// there should also be considered an error.
	RemoveLinkToken(linkToken string) error
}

// ------------------------------------------------------------------------------

func createKey(namespace, email string) string {
	return fmt.Sprintf("%s:token:%s", namespace, email)
}

func createLinkKey(namespace, linkToken string) string {
	return fmt.Sprintf("%s:linktoken:%s", namespace, linkToken)
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

func (s *RedisTokenStorage) StoreLinkToken(linkToken, email string) error {
	ctx := context.Background()
	return s.client.Set(ctx, createLinkKey(s.namespace, linkToken), email, Timeout).Err()
}

func (s *RedisTokenStorage) RetrieveEmailByLinkToken(linkToken string) (string, error) {
	ctx := context.Background()
	return s.client.Get(ctx, createLinkKey(s.namespace, linkToken)).Result()
}

func (s *RedisTokenStorage) RemoveLinkToken(linkToken string) error {
	ctx := context.Background()
	return s.client.Del(ctx, createLinkKey(s.namespace, linkToken)).Err()
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

func (s *InMemoryTokenStorage) StoreLinkToken(linkToken, email string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.LinkTokenMap[linkToken] = email
	return nil
}

func (s *InMemoryTokenStorage) RetrieveEmailByLinkToken(linkToken string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if email, ok := s.LinkTokenMap[linkToken]; ok {
		return email, nil
	} else {
		return "", fmt.Errorf("failed to find email for link token")
	}
}

func (s *InMemoryTokenStorage) RemoveLinkToken(linkToken string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.LinkTokenMap[linkToken]; ok {
		delete(s.LinkTokenMap, linkToken)
		return nil
	} else {
		return fmt.Errorf("failed to remove link token, because it wasn't there")
	}
}
