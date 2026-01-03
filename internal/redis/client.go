// Package redis provides Redis client utilities for Quasar.
package redis

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis client with convenience methods
type Client struct {
	*redis.Client
}

// ParseRedisURL parses a redis:// URL and returns options
func ParseRedisURL(rawURL string) (*redis.Options, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty Redis URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	opts := &redis.Options{
		Addr: u.Host,
	}

	// Default port if not specified
	if u.Port() == "" {
		opts.Addr = u.Hostname() + ":6379"
	}

	// Password from URL
	if u.User != nil {
		if pwd, ok := u.User.Password(); ok {
			opts.Password = pwd
		}
	}

	// Database from path (e.g., redis://localhost/1)
	if len(u.Path) > 1 {
		dbStr := u.Path[1:] // Remove leading slash
		if db, err := strconv.Atoi(dbStr); err == nil {
			opts.DB = db
		}
	}

	return opts, nil
}

// NewClient creates a new Redis client from URL
func NewClient(redisURL string) (*Client, error) {
	opts, err := ParseRedisURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// NewClientLazy creates a client without testing connection
func NewClientLazy(redisURL string) (*Client, error) {
	opts, err := ParseRedisURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &Client{Client: redis.NewClient(opts)}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.Client.Close()
}
