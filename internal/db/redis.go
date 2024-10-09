package database

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

// Cache struct with Redis client
type Cache struct {
	Redis *redis.Client
	Ctx   context.Context
}

func RedisClient(address string) *Cache {
	return &Cache{
		Redis: redis.NewClient(&redis.Options{
			Addr: address,
		}),
		Ctx: context.Background(),
	}
}

func (c *Cache) SaveAddress(addr string) error {
	err := c.Redis.SAdd(c.Ctx, "ethAddresses", addr).Err() // Use SAdd to add to a set
	if err != nil {
		return fmt.Errorf("failed to save address to Redis: %v", err)
	}
	fmt.Println("Address saved:", addr)
	return nil
}

func (c *Cache) DeleteAddress(addr string) error {
	err := c.Redis.SRem(c.Ctx, "ethAddresses", addr).Err() // Use SRem to remove from the set
	if err != nil {
		return fmt.Errorf("failed to delete address from Redis: %v", err)
	}
	fmt.Println("Address deleted:", addr)
	return nil
}

func (c *Cache) GetAllAddresses() ([]string, error) {
	var addresses []string

	// Get all addresses from the Redis set
	addrStr, err := c.Redis.SMembers(c.Ctx, "ethAddresses").Result()
	if err != nil {
		return nil, fmt.Errorf("error retrieving addresses from Redis: %v", err)
	}

	for _, addrStr := range addrStr {
		address := addrStr
		addresses = append(addresses, address)
	}

	return addresses, nil
}
