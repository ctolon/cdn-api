package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient is a Base Redis client for the application
type RedisClient struct {
	Client  *redis.Client
	Options *redis.Options
	mu      *sync.RWMutex
}

// NewRedisClientFromOptions creates a new Redis client from options
func NewRedisClientFromOptions(options *redis.Options) *RedisClient {
	client := redis.NewClient(options)
	r := &RedisClient{
		Client:  client,
		Options: options,
		mu:      &sync.RWMutex{},
	}
	return r
}

// NewRedisClientFromURL creates a new Redis client from URL
func NewRedisClientFromURL(url string) *RedisClient {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(opt)
	r := &RedisClient{
		Client:  client,
		Options: opt,
		mu:      &sync.RWMutex{},
	}
	return r
}

// NewRedisClientFromURLWithDb creates a new Redis client from URL with DB
func NewRedisClientFromURLWithDb(url string, db string) *RedisClient {
	opt, err := redis.ParseURL(url + "/" + db + "?" + "read_timeout=-1")
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(opt)
	r := &RedisClient{
		Client:  client,
		Options: opt,
		mu:      &sync.RWMutex{},
	}
	return r
}

// NewRedisClientFromURLWithDbAndPool creates a new Redis client with specified options
func NewRedisClientFromURLWithDbAndPool(url string, db int, poolSize int) *RedisClient {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	// Set the DB and PoolSize in the options
	opt.DB = db
	opt.PoolSize = poolSize

	client := redis.NewClient(opt)

	r := &RedisClient{
		Client:  client,
		Options: opt,
		mu:      &sync.RWMutex{},
	}

	return r
}

// Get gets a key from Redis
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		errMsg := fmt.Errorf("could not get key %s: %v", key, err)
		return "", errMsg
	}
	return val, nil
}

// Set sets a key in Redis
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}) error {
	_, err := r.Client.Set(ctx, key, value, 0).Result()
	if err != nil {
		return fmt.Errorf("could not set key %s: %w", key, err)
	}
	return nil
}

// SetWithTTL sets a key in Redis with an expiration time
func (r *RedisClient) SetWithTTL(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	_, err := r.Client.Set(ctx, key, value, expiration).Result()
	if err != nil {
		return fmt.Errorf("could not set key %s: %w", key, err)
	}
	return nil
}

// Del deletes a key from Redis
func (r *RedisClient) Del(ctx context.Context, key string) error {
	_, err := r.Client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("could not delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in Redis
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	val, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("could not check if key exists: %w", err)
	}
	return val == 1, nil
}

// Publish publishes a message to a channel in Redis
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	_, err := r.Client.Publish(ctx, channel, message).Result()
	if err != nil {
		return fmt.Errorf("could not publish message to channel %s: %w", channel, err)
	}
	return nil
}

// CreateSubscribeChannel creates a new subscription channel in Redis
func (r *RedisClient) CreateSubscribeChannel(ctx context.Context, channel string) *redis.PubSub {
	pubsub := r.Client.Subscribe(ctx, channel)
	//if _, err := pubsub.Receive(ctx); err != nil {
	//    rootLogger.Error("failed to receive from control PubSub", zap.Error(err))
	//    return nil
	//}
	return pubsub
}

// StartSubscribe starts listening to a subscription channel in Redis
func (r *RedisClient) StartSubscribe(ctx context.Context, ch <-chan *redis.Message) {
	for msg := range ch {
		fmt.Println(msg.Channel, msg.Payload)
	}
}

// Close closes the Redis client
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// Ping pings the Redis server
func (r *RedisClient) Ping(ctx context.Context) error {
	_, err := r.Client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("could not ping Redis server: %w", err)
	}
	return nil
}

// XAdd adds values to a stream in Redis
func (r *RedisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	_, err := r.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	if err != nil {
		return fmt.Errorf("could not add values to stream %s: %w", stream, err)
	}
	return nil
}

// XRead reads values from a stream in Redis
func (r *RedisClient) XRead(ctx context.Context, streams []string, count int64, block time.Duration) ([]redis.XStream, error) {
	val, err := r.Client.XRead(ctx, &redis.XReadArgs{
		Streams: streams,
		Count:   count,
		Block:   block,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("could not read from streams: %w", err)
	}
	return val, nil
}

// XGroupCreate creates a consumer group in Redis
func (r *RedisClient) XGroupCreate(ctx context.Context, stream, group, start string) error {
	if start == "" {
		start = ">"
	}
	if err := r.Client.XGroupCreate(ctx, stream, group, start).Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("could not create consumer group: %w", err)
	}
	return nil
}

// XGroupCreateMkStream creates a consumer group in Redis with a stream
func (r *RedisClient) XGroupCreateMkStream(ctx context.Context, stream, group, start string) error {
	if start == "" {
		start = "$"
	}
	if err := r.Client.XGroupCreateMkStream(ctx, stream, group, start).Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("could not create consumer group with stream: %w", err)
	}
	return nil
}

// XAck acknowledges a message in a stream in Redis
func (r *RedisClient) XAck(ctx context.Context, stream, group string, ids ...string) error {
	_, err := r.Client.XAck(ctx, stream, group, ids...).Result()
	if err != nil {
		return fmt.Errorf("could not acknowledge message: %w", err)
	}
	return nil
}

// XReadGroup reads messages from a stream in a consumer group in Redis
func (r *RedisClient) XReadGroup(ctx context.Context, group, consumer, stream string, count int64, block time.Duration) ([]redis.XStream, error) {
	val, err := r.Client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("could not read from group: %w", err)
	}
	return val, nil
}

// XClaim claims messages from a stream in a consumer group in Redis
func (r *RedisClient) XClaim(ctx context.Context, args *redis.XClaimArgs) ([]redis.XMessage, error) {
	val, err := r.Client.XClaim(ctx, args).Result()
	if err != nil {
		return nil, fmt.Errorf("could not claim messages: %w", err)
	}
	return val, nil
}

// XTrimMaxID trims a stream in Redis
func (r *RedisClient) XTrimMinID(ctx context.Context, stream string, minID string) error {
	_, err := r.Client.XTrimMinID(ctx, stream, minID).Result()
	if err != nil {
		return fmt.Errorf("could not trim stream: %w", err)
	}
	return nil
}

// XDel deletes a message from a stream in Redis
func (r *RedisClient) XDel(ctx context.Context, stream string, ids ...string) error {
	_, err := r.Client.XDel(ctx, stream, ids...).Result()
	if err != nil {
		return fmt.Errorf("could not delete message: %w", err)
	}
	return nil
}

// XLen gets the length of a stream in Redis
func (r *RedisClient) XLen(ctx context.Context, stream string) (int64, error) {
	val, err := r.Client.XLen(ctx, stream).Result()
	if err != nil {
		return 0, fmt.Errorf("could not get length of stream: %w", err)
	}
	return val, nil
}

// XRange gets a range of messages from a stream in Redis
func (r *RedisClient) XRange(ctx context.Context, stream, start, stop string) ([]redis.XMessage, error) {
	val, err := r.Client.XRange(ctx, stream, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("could not get range of messages: %w", err)
	}
	return val, nil
}
