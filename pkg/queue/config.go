package queue

import (
	"fmt"
	"time"
)

// AckMode controls when a delivered message is acknowledged.
type AckMode string

const (
	AckOnSuccess    AckMode = "on_success"
	AckBeforeHandle AckMode = "before_handle"
	AckAfterHandle  AckMode = "after_handle"
)

const (
	defaultGroup           = "workers"
	defaultConcurrency     = 1
	defaultBlockTimeout    = time.Second
	defaultReclaimInterval = 10 * time.Second
	defaultReclaimIdle     = time.Minute
	defaultStartID         = "0"
)

// Config selects one Redis connection and transport settings for named queues.
type Config struct {
	Redis    RedisConfig         `yaml:"redis"`
	Defaults Settings            `yaml:"defaults"`
	Queues   map[string]Settings `yaml:"queues"`
}

type RedisConfig struct {
	PoolName string `yaml:"pool_name"`
	Prefix   string `yaml:"prefix"`
}

// Settings contains transport settings. Zero fields inherit defaults.
// Rate limits and business retries belong to the callback.
type Settings struct {
	Group           string        `yaml:"group"`
	AckMode         AckMode       `yaml:"ack_mode"`
	Concurrency     int           `yaml:"concurrency"`
	BatchSize       int           `yaml:"batch_size"`
	BlockTimeout    time.Duration `yaml:"block_timeout"`
	HandlerTimeout  time.Duration `yaml:"handler_timeout"`
	ReclaimInterval time.Duration `yaml:"reclaim_interval"`
	ReclaimIdle     time.Duration `yaml:"reclaim_idle"`
	StartID         string        `yaml:"start_id"`
}

func NewConfig() Config {
	return Config{
		Defaults: Settings{
			Group:           defaultGroup,
			AckMode:         AckOnSuccess,
			Concurrency:     defaultConcurrency,
			BlockTimeout:    defaultBlockTimeout,
			ReclaimInterval: defaultReclaimInterval,
			ReclaimIdle:     defaultReclaimIdle,
			StartID:         defaultStartID,
		},
		Queues: make(map[string]Settings),
	}
}

// Prepare normalizes and validates the transport settings.
func (c Config) Prepare() (Config, error) {
	batchSize := c.Defaults.BatchSize
	c.Defaults.Normalize(NewConfig().Defaults)
	if err := c.Defaults.validate(); err != nil {
		return Config{}, err
	}
	defaults := c.Defaults
	defaults.BatchSize = batchSize
	queues := make(map[string]Settings, len(c.Queues))
	for name, cfg := range c.Queues {
		if name == "" {
			return Config{}, fmt.Errorf("queue name is required")
		}
		cfg.Normalize(defaults)
		if err := cfg.validate(); err != nil {
			return Config{}, fmt.Errorf("queue %s: %w", name, err)
		}
		queues[name] = cfg
	}

	c.Queues = queues

	return c, nil
}

func (c Settings) validate() error {
	if c.AckMode != AckOnSuccess && c.AckMode != AckBeforeHandle && c.AckMode != AckAfterHandle {
		return fmt.Errorf("invalid ack_mode %q", c.AckMode)
	}
	if min(c.Concurrency, c.BatchSize) < 1 {
		return fmt.Errorf("concurrency and batch_size must be positive")
	}
	if min(c.BlockTimeout, c.ReclaimInterval) < time.Millisecond || c.ReclaimIdle < 3*time.Millisecond ||
		c.HandlerTimeout < 0 {
		return fmt.Errorf("invalid queue timeouts or intervals")
	}

	return nil
}

// Normalize fills unset fields from defaults.
//
//nolint:cyclop // Keep one explicit default assignment per field.
func (c *Settings) Normalize(defaults Settings) {
	if c.Group == "" {
		c.Group = defaults.Group
	}
	if c.AckMode == "" {
		c.AckMode = defaults.AckMode
	}
	if c.Concurrency == 0 {
		c.Concurrency = defaults.Concurrency
	}
	if c.BatchSize == 0 {
		c.BatchSize = defaults.BatchSize
		if c.BatchSize == 0 {
			c.BatchSize = c.Concurrency
		}
	}
	if c.BlockTimeout == 0 {
		c.BlockTimeout = defaults.BlockTimeout
	}
	if c.HandlerTimeout == 0 {
		c.HandlerTimeout = defaults.HandlerTimeout
	}
	if c.ReclaimInterval == 0 {
		c.ReclaimInterval = defaults.ReclaimInterval
	}
	if c.ReclaimIdle == 0 {
		c.ReclaimIdle = defaults.ReclaimIdle
	}
	if c.StartID == "" {
		c.StartID = defaults.StartID
	}
}
