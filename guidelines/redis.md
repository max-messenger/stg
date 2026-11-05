# Redis

Доступ выполняется через локальный `stg/pkg/connections/rediscli`. Именованные
pools задаются под `redis:`; используйте имя из конфигурации приложения.

```yaml
redis:
  main:
    addrs: ["localhost:6379"]
    key_prefix: "stg"
```

`GetPool` вызывается один раз при построении repository. Все операции выполняйте
внутри `Call(ctx, operation, callback)`, keys стройте через `kf.FormatKey`.
Formatter добавляет pool prefix и разделитель; не добавляйте prefix повторно.
Для Sentinel/cluster используйте поддерживаемые поля текущего
[Config](../pkg/connections/rediscli/config.go), а не отдельный raw client.

## Пример repository

```go
package cacherepo

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "stg/pkg/connections/rediscli"
)

var ErrNotFound = errors.New("cache entry not found")

type Repository struct {
    pool *rediscli.Redis
}

func New(pools *rediscli.Pool) (*Repository, error) {
    pool, err := pools.GetPool("main")
    if err != nil {
        return nil, fmt.Errorf("get cache pool: %w", err)
    }
    return &Repository{pool: pool}, nil
}

func (r *Repository) Get(ctx context.Context, key string) (string, error) {
    var value string
    found := false
    err := r.pool.Call(ctx, "get_cache_entry", func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
        result, qerr := ucl.Get(ctx, kf.FormatKey(key)).Result()
        if errors.Is(qerr, redis.Nil) {
            return nil
        }
        if qerr != nil {
            return fmt.Errorf("get cache entry: %w", qerr)
        }
        value, found = result, true
        return nil
    })
    if err != nil {
        return "", fmt.Errorf("read cache: %w", err)
    }
    if !found {
        return "", ErrNotFound
    }
    return value, nil
}

func (r *Repository) Set(ctx context.Context, key, value string, ttl time.Duration) error {
    err := r.pool.Call(ctx, "set_cache_entry", func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error {
        if qerr := ucl.Set(ctx, kf.FormatKey(key), value, ttl).Err(); qerr != nil {
            return fmt.Errorf("set cache entry: %w", qerr)
        }
        return nil
    })
    if err != nil {
        return fmt.Errorf("write cache: %w", err)
    }
    return nil
}
```

`ErrNotFound` здесь локальный демонстрационный error; бизнес-ошибки приложения
размещайте в `internal/app/domain`. Redis miss не должен выходить из callback
как техническая ошибка и портить tracing/metrics.

Structured values кодируйте через `stg/pkg/marshaler`. TTL задавайте для данных
с ограниченным временем жизни; постоянные данные не требуют произвольного TTL.
Repository module провайдит конкретную реализацию; owning service объявляет
локальный интерфейс и Fx-адаптер.

Источники: [wrapper](../pkg/connections/rediscli/redis.go),
[pool](../pkg/connections/rediscli/pool.go). Для multi-step Redis operations
проверяйте atomicity выбранных команд/transactions отдельно.
