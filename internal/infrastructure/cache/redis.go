package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps go-redis com helpers simples de Get/Set/Del.
// Todos os métodos são seguros para ponteiro nil — retornam zero sem erro.
type Cache struct {
	client *redis.Client
}

// ConfigFromEnv lê REDIS_URL (ex: "redis://localhost:6379").
func ConfigFromEnv() string {
	return os.Getenv("REDIS_URL")
}

// New conecta ao Redis e valida a conexão via PING.
func New(redisURL string) (*Cache, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("cache: REDIS_URL não configurado")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("cache: parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache: ping Redis: %w", err)
	}
	return &Cache{client: client}, nil
}

// Get busca uma chave. Retorna ("", nil) se ausente ou se cache for nil.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if c == nil {
		return "", nil
	}
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Set armazena key=value com TTL. No-op se cache for nil.
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Del remove uma chave. No-op se cache for nil.
func (c *Cache) Del(ctx context.Context, key string) error {
	if c == nil {
		return nil
	}
	return c.client.Del(ctx, key).Err()
}
