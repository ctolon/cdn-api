package main

import (
	"context"
	//"os"
	// "sync"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/config"
	"github.com/ctolon/cdn-api/internal/core/cache"
	//"github.com/rs/zerolog"
)

func main() {

	cfgProvider := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(cfgProvider)

	client := cache.NewRedisClientFromURL(cfg.RedisURL)

	// Publish a message to a Redis Stream
	err := client.XAdd(context.Background(), "redis-master", map[string]interface{}{"message": "Hello, World!"})
	if err != nil {
		panic(err)
	}
}
