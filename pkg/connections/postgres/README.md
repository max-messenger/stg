# Postgres

Клиент для postgres на основе `pgx/v4`

## Использование

Подключить можно через `fx`

```go
fx.Options(
    postgres.Module,
)
```

После в сервисе вызываем получение коннекта к postgre

```go
func NewService(postgresPool *postgres.Pool) (*Service, error) {
    connect, _ := postgresPool.Get("some_service")
    
    return &Service{
        connect: connect,
    }
}
```

Дальше уже можно использовать коннект для работы с postgre

```go
func (s *Service) DoSomething(ctx context.Context) error {
    return s.connect.Call(ctx, "call_name", func(ctx context.Context, pool postgres.Queryable) error {
        return clu.Exec(ctx, "some_query").Err()
    })
}
```

## Коллбеки перед и после соединения

Можно передать коллбеки для перед и после соединения через группу `pg_callbacks`

Требуется например для регистрации парсинга типов.

```go
func (r *Repo) AfterConnectCallback(_ context.Context, c *pgx.Conn) error {
    tm := c.TypeMap()
    tm.RegisterDefaultPgType(uuid.NullUUID{}, "uuid")

    return nil
}
```
