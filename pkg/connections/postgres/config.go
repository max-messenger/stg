package postgres

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

type PoolConfig map[string]*Config

type Config struct {
	DatabaseURL string        `yaml:"database_url"`
	Backoff     BackoffConfig `yaml:"backoff"`
}

type BackoffConfig struct {
	Enabled bool `yaml:"enabled"`

	MaxRetries *uint64 `yaml:"max_retries"`

	InitialInterval     *time.Duration `yaml:"initial_interval"`
	MaxInterval         *time.Duration `yaml:"max_interval"`
	MaxElapsedTime      *time.Duration `yaml:"max_elapsed_time"`
	Multiplier          *float64       `yaml:"multiplier"`
	RandomizationFactor *float64       `yaml:"randomization_factor"`
}

func (c BackoffConfig) NewBackOff() func() backoff.BackOff {
	if !c.Enabled {
		return func() backoff.BackOff { return nil }
	}

	var opts []backoff.ExponentialBackOffOpts

	if c.InitialInterval != nil {
		opts = append(opts, backoff.WithInitialInterval(*c.InitialInterval))
	}

	if c.MaxInterval != nil {
		opts = append(opts, backoff.WithMaxInterval(*c.MaxInterval))
	}

	if c.MaxElapsedTime != nil {
		opts = append(opts, backoff.WithMaxElapsedTime(*c.MaxElapsedTime))
	}

	if c.Multiplier != nil {
		opts = append(opts, backoff.WithMultiplier(*c.Multiplier))
	}

	if c.RandomizationFactor != nil {
		opts = append(opts, backoff.WithRandomizationFactor(*c.RandomizationFactor))
	}

	return func() backoff.BackOff {
		bo := backoff.BackOff(backoff.NewExponentialBackOff(opts...))
		if c.MaxRetries != nil {
			bo = backoff.WithMaxRetries(bo, *c.MaxRetries)
		}

		return bo
	}
}
