package tokenrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"stg/internal/app/domain"
	"stg/pkg/connections/rediscli"
	"stg/pkg/marshaler"
)

// Store интерфейс для работы с Redis.
type Store interface {
	Call(ctx context.Context, target string, f rediscli.CallCallback) error
}

// Repository управляет хранением refresh-токенов в Redis.
type Repository struct {
	store      Store
	refreshTTL time.Duration
}

// NewRepository создаёт экземпляр Repository.
func NewRepository(store Store, cfg Config) *Repository {
	return &Repository{
		store:      store,
		refreshTTL: cfg.RefreshTTL,
	}
}

func (r *Repository) StoreRefresh(ctx context.Context, data domain.AuthData) error {
	raw, err := marshaler.MarshalJSON(data)
	if err != nil {
		return fmt.Errorf("marshal refresh data: %w", err)
	}

	return r.store.Call(ctx, "store_refresh", func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
		key := kf.FormatKey(r.serializeKey(data.User.ID))

		return ucl.Set(ctx, key, raw, r.refreshTTL).Err()
	})
}

// GetRefresh получает данные refresh token.
func (r *Repository) GetRefresh(ctx context.Context, userID int64) (domain.AuthData, error) {
	var result domain.AuthData

	err := r.store.Call(ctx, "get_refresh", func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
		key := kf.FormatKey(r.serializeKey(userID))

		raw, err := ucl.Get(ctx, key).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}

		if err == nil {
			return marshaler.UnmarshalJSONFromString(raw, &result)
		}

		return err
	})
	if err != nil {
		return result, fmt.Errorf("get refresh: %w", err)
	}

	return result, nil
}

// DeleteRefresh удаляет refresh token (logout / rotation).
func (r *Repository) DeleteRefresh(ctx context.Context, userID int64) error {
	return r.store.Call(ctx, "delete_refresh", func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
		key := kf.FormatKey(r.serializeKey(userID))

		return ucl.Del(ctx, key).Err()
	})
}

func (r *Repository) serializeKey(userID int64) string {
	return fmt.Sprintf("refresh:%d", userID)
}
