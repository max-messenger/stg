package staterepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"stg/pkg/connections/rediscli"
)

const (
	domainKey  = "state"
	defaultTTL = time.Hour

	traceGet = "state-pop"
	traceSet = "state-set"
)

type Store interface {
	Call(ctx context.Context, target string, f rediscli.CallCallback) error
}

type Repository struct {
	store Store
}

func NewRepository(store Store) *Repository {
	return &Repository{
		store: store,
	}
}

func (r *Repository) GetState(ctx context.Context, userID int64) (string, error) {
	var value string
	cErr := r.store.Call(ctx, traceGet, func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
		cmd := ucl.GetDel(ctx, kf.FormatKey(r.serializeKey(userID)))
		err := cmd.Err()

		value = cmd.Val()
		if err == nil && value != "" {
			return nil
		}

		if err != nil {
			if errors.Is(err, redis.Nil) {
				err = nil
			}

			return err
		}

		return nil
	})
	if cErr != nil {
		return "", fmt.Errorf("state repository get: %w", cErr)
	}

	return value, nil
}

func (r *Repository) SetState(ctx context.Context, userID int64, value string) error {
	err := r.store.Call(ctx, traceSet, func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
		return ucl.Set(ctx, kf.FormatKey(r.serializeKey(userID)), value, defaultTTL).Err()
	})
	if err != nil {
		return fmt.Errorf("state repository set: %w", err)
	}

	return nil
}

func (r *Repository) ClearState(ctx context.Context, userID int64) error {
	err := r.store.Call(ctx, traceSet, func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
		return ucl.Del(ctx, kf.FormatKey(r.serializeKey(userID))).Err()
	})
	if err != nil {
		return fmt.Errorf("state repository clear: %w", err)
	}

	return nil
}

func (r *Repository) serializeKey(userID int64) string {
	return fmt.Sprintf("%s:%d", domainKey, userID)
}
