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
// Timeout curto evita que Redis pendurado bloqueie a request inteira.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if c == nil {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Set armazena key=value com TTL. No-op se cache for nil.
// Timeout curto evita que Redis pendurado bloqueie a request inteira.
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Del remove uma chave. No-op se cache for nil.
func (c *Cache) Del(ctx context.Context, key string) error {
	if c == nil {
		return nil
	}
	return c.client.Del(ctx, key).Err()
}

// Incr incrementa uma chave inteira e, na primeira criação, aplica o TTL
// (contadores mensais expiram sozinhos ao virar o mês). Retorna o novo valor.
// Se o cache for nil, retorna (0, nil) — sem Redis não há metering (fail-open).
func (c *Cache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if c == nil {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	n, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 && ttl > 0 {
		_ = c.client.Expire(ctx, key, ttl).Err()
	}
	return n, nil
}

// Decr decrementa uma chave inteira (usado para devolver cota reservada quando a
// operação subsequente falha). No-op se cache for nil.
func (c *Cache) Decr(ctx context.Context, key string) error {
	if c == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	return c.client.Decr(ctx, key).Err()
}

// GetInt lê o valor inteiro de uma chave. Retorna (0, nil) se ausente ou nil.
func (c *Cache) GetInt(ctx context.Context, key string) (int64, error) {
	if c == nil {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	n, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return n, err
}
