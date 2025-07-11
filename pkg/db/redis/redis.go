package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	log "github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/rs/zerolog"
)

// KVStore defines the interface for key-value store operations
type KVStore interface {
	Close() error
	SafeFlushPattern(pattern string) error
	GetJSON(key string, out interface{}) error
	SetJSON(key string, value interface{}, ttl time.Duration) error
	Exists(key string) (bool, error)
	Del(key string) error
}

// KVStoreWithRetries defines the interface for key-value store operations with retry functionality
type KVStoreWithRetries interface {
	KVStore
	SafeFlushPatternWithMaxRetries(pattern string, maxRetries int) error
	GetJSONWithMaxRetries(key string, out interface{}, maxRetries int) error
	SetJSONWithMaxRetries(key string, value interface{}, ttl time.Duration, maxRetries int) error
	ExistsWithMaxRetries(key string, maxRetries int) (bool, error)
	DelWithMaxRetries(key string, maxRetries int) error
}

// LockManager defines the interface for distributed locking operations
type LockManager interface {
	AcquireLock(lockKey string, lockValue interface{}, ttl time.Duration) (bool, error)
	ReleaseLock(lockKey string) error
}

// LockManagerWithRetries defines the interface for distributed locking operations with retry functionality
type LockManagerWithRetries interface {
	LockManager
	AcquireLockWithMaxRetries(lockKey string, lockValue interface{}, ttl time.Duration, maxRetries int) (bool, error)
	ReleaseLockWithMaxRetries(lockKey string, maxRetries int) error
}

// RedisStore combines both KVStore and LockManager interfaces
type RedisStore interface {
	KVStore
	LockManager
	KVStoreWithRetries
	LockManagerWithRetries
}

// RedisStoreWithRetries combines both KVStoreWithRetries and LockManagerWithRetries interfaces
type RedisStoreWithRetries interface {
	KVStoreWithRetries
	LockManagerWithRetries
}

// RedisClient implements the RedisStoreWithRetries interface
type RedisClient struct {
	client *redis.Client
	logger *log.Logger
}

// NewRedisClient creates a new Redis client from a URI
func NewRedisClient(redisAddr, redisPassword string, redisDB int) (RedisStoreWithRetries, error) {
	logger := log.NewLogger(nil, zerolog.InfoLevel, "redis_store")

	options := &redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}

	client := redis.NewClient(options)
	if err := client.Ping().Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}

	logger.LogInfoMessage().Msg("successfully connected to redis")

	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (c *RedisClient) Close() error {
	return c.client.Close()
}

// SafeFlushPattern deletes all keys matching a pattern
func (c *RedisClient) SafeFlushPattern(pattern string) error {
	var cursor uint64
	var keys []string
	var err error

	for {
		keys, cursor, err = c.client.Scan(cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("error scanning keys: %v", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(keys...).Err(); err != nil {
				return fmt.Errorf("error deleting keys: %v", err)
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}

// SafeFlushPatternWithMaxRetries deletes all keys matching a pattern with retry logic
func (c *RedisClient) SafeFlushPatternWithMaxRetries(pattern string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("SafeFlushPattern attempt %d/%d for pattern %s", attempt, maxRetries, pattern)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		err := c.SafeFlushPattern(pattern)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("SafeFlushPattern attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("SafeFlushPattern failed after %d retries: %v", maxRetries, lastErr)
}

// GetJSON retrieves a JSON value from Redis and unmarshals it
func (c *RedisClient) GetJSON(key string, out interface{}) error {
	data, err := c.client.Get(key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found")
		}
		return fmt.Errorf("failed to get key %s: %v", key, err)
	}
	return json.Unmarshal(data, out)
}

// GetJSONWithMaxRetries retrieves a JSON value from Redis with retry logic
func (c *RedisClient) GetJSONWithMaxRetries(key string, out interface{}, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("GetJSON attempt %d/%d for key %s", attempt, maxRetries, key)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		err := c.GetJSON(key, out)
		if err == nil {
			return nil
		}

		// If key not found, don't retry
		if err.Error() == "key not found" {
			return err
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("GetJSON attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("GetJSON failed after %d retries: %v", maxRetries, lastErr)
}

// SetJSON marshals a value to JSON and stores it in Redis
func (c *RedisClient) SetJSON(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}
	return c.client.Set(key, data, ttl).Err()
}

// SetJSONWithMaxRetries marshals a value to JSON and stores it in Redis with retry logic
func (c *RedisClient) SetJSONWithMaxRetries(key string, value interface{}, ttl time.Duration, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("SetJSON attempt %d/%d for key %s", attempt, maxRetries, key)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		err := c.SetJSON(key, value, ttl)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("SetJSON attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("SetJSON failed after %d retries: %v", maxRetries, lastErr)
}

// Exists checks if a key exists
func (c *RedisClient) Exists(key string) (bool, error) {
	count, err := c.client.Exists(key).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("error checking existence: %v", err)
	} else if err == redis.Nil {
		return false, nil
	}
	return count > 0, nil
}

// ExistsWithMaxRetries checks if a key exists with retry logic
func (c *RedisClient) ExistsWithMaxRetries(key string, maxRetries int) (bool, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("Exists attempt %d/%d for key %s", attempt, maxRetries, key)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		exists, err := c.Exists(key)
		if err == nil {
			return exists, nil
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("Exists attempt %d failed: %v", attempt+1, err)
	}

	return false, fmt.Errorf("Exists failed after %d retries: %v", maxRetries, lastErr)
}

// Del deletes a key
func (c *RedisClient) Del(key string) error {
	return c.client.Del(key).Err()
}

// DelWithMaxRetries deletes a key with retry logic
func (c *RedisClient) DelWithMaxRetries(key string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("Del attempt %d/%d for key %s", attempt, maxRetries, key)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		err := c.Del(key)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("Del attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("Del failed after %d retries: %v", maxRetries, lastErr)
}

// AcquireLock attempts to acquire a distributed lock
func (c *RedisClient) AcquireLock(lockKey string, lockValue interface{}, ttl time.Duration) (bool, error) {
	lock, err := c.client.SetNX(lockKey, lockValue, ttl).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	return lock, nil
}

// AcquireLockWithMaxRetries attempts to acquire a distributed lock with retry logic
func (c *RedisClient) AcquireLockWithMaxRetries(lockKey string, lockValue interface{}, ttl time.Duration, maxRetries int) (bool, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("AcquireLock attempt %d/%d for lock %s", attempt, maxRetries, lockKey)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		acquired, err := c.AcquireLock(lockKey, lockValue, ttl)
		if err == nil {
			return acquired, nil
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("AcquireLock attempt %d failed: %v", attempt+1, err)
	}

	return false, fmt.Errorf("AcquireLock failed after %d retries: %v", maxRetries, lastErr)
}

// ReleaseLock releases a distributed lock
func (c *RedisClient) ReleaseLock(lockKey string) error {
	return c.Del(lockKey)
}

// ReleaseLockWithMaxRetries releases a distributed lock with retry logic
func (c *RedisClient) ReleaseLockWithMaxRetries(lockKey string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.LogWarnMessage().Msgf("ReleaseLock attempt %d/%d for lock %s", attempt, maxRetries, lockKey)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // exponential backoff
		}

		err := c.ReleaseLock(lockKey)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.LogErrorMessage().Msgf("ReleaseLock attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("ReleaseLock failed after %d retries: %v", maxRetries, lastErr)
}
