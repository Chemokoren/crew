package rolecache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisAdapter wraps *redis.Client to satisfy the RedisStore interface.
type RedisAdapter struct {
	client *redis.Client
}

// NewRedisAdapter creates a RedisStore backed by a real Redis client.
func NewRedisAdapter(client *redis.Client) *RedisAdapter {
	return &RedisAdapter{client: client}
}

func (a *RedisAdapter) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return a.client.Get(ctx, key).Bytes()
}

func (a *RedisAdapter) SetBytes(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

// RedisPubSubAdapter wraps *redis.Client to satisfy the PubSubSubscriber interface.
type RedisPubSubAdapter struct {
	client *redis.Client
}

// NewRedisPubSubAdapter creates a PubSubSubscriber backed by a real Redis client.
func NewRedisPubSubAdapter(client *redis.Client) *RedisPubSubAdapter {
	return &RedisPubSubAdapter{client: client}
}

// Subscribe subscribes to a Redis Pub/Sub channel and returns a message channel.
// The cancel function must be called to unsubscribe and clean up.
func (a *RedisPubSubAdapter) Subscribe(ctx context.Context, channel string) (<-chan string, func(), error) {
	sub := a.client.Subscribe(ctx, channel)

	// Verify the subscription is active
	_, err := sub.Receive(ctx)
	if err != nil {
		sub.Close()
		return nil, nil, err
	}

	msgChan := make(chan string, 16) // Buffered to prevent blocking publisher
	cancel := func() {
		sub.Close()
		close(msgChan)
	}

	go func() {
		ch := sub.Channel()
		for msg := range ch {
			select {
			case msgChan <- msg.Payload:
			default:
				// Drop message if buffer is full (non-critical; next cron will catch up)
			}
		}
	}()

	return msgChan, cancel, nil
}
