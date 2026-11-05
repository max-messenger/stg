# PostgreSQL

Используйте локальный `stg/pkg/connections/postgres` и именованные pools.
В исходном примере `stg-sel` предназначен для reads, `stg-upd` — для mutations.
После `create.sh` эти имена переименовываются вместе с приложением.

## Конфигурация

```yaml
postgres:
  stg-sel:
    database_url: ${POSTGRES_CONNECTION_STRING_SEL}
  stg-upd:
    database_url: ${POSTGRES_CONNECTION_STRING_UPD}
```

Используйте DSN/query execution mode, совместимый с SQL placeholders и
PgBouncer вашего окружения. Пример STG строит запросы `go-sqlbuilder` и использует
`default_query_exec_mode=exec`; для конкретного PgBouncer может потребоваться
`simple_protocol`. Не переносите настройки без проверки deployment/config.

Конструктор один раз получает read/write pools через `GetPool`. Repository
module провайдит конкретный тип, owning service адаптирует его к локальному
интерфейсу. Table/column names определяйте constants; пользовательские значения
передавайте параметрами, а не интерполяцией в SQL.

## Пример чтения

Ниже отдельный repository пример с закрытием rows и проверкой `rows.Err()`:

```go
package itemsrepo

import (
    "context"
    "fmt"

    "github.com/huandu/go-sqlbuilder"
    "stg/pkg/connections/postgres"
)

const (
    tableName = "items"
    fieldID = "id"
)

type Repository struct {
    read *postgres.Postgres
}

func New(pool *postgres.Pool) (*Repository, error) {
    read, err := pool.GetPool("stg-sel")
    if err != nil {
        return nil, fmt.Errorf("get read pool: %w", err)
    }
    return &Repository{read: read}, nil
}

func (r *Repository) ListIDs(ctx context.Context) ([]int64, error) {
    sb := sqlbuilder.Select(fieldID).From(tableName)
    query, args := sb.Build()
    var ids []int64
    err := r.read.Call(ctx, "list_item_ids", func(ctx context.Context, db postgres.Queryable) error {
        ids = nil // Call может повторить callback при включённом backoff.
        rows, qerr := db.Query(ctx, query, args...)
        if qerr != nil {
            return fmt.Errorf("query item ids: %w", qerr)
        }
        defer rows.Close()
        for rows.Next() {
            var id int64
            if qerr := rows.Scan(&id); qerr != nil {
                return fmt.Errorf("scan item id: %w", qerr)
            }
            ids = append(ids, id)
        }
        if qerr := rows.Err(); qerr != nil {
            return fmt.Errorf("iterate item ids: %w", qerr)
        }
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("list item ids: %w", err)
    }
    return ids, nil
}
```

Для записи используйте write pool и `sqlbuilder.InsertInto`, `Update` или
`DeleteFrom` по существующим API. В `Update.Set` передавайте готовые assignments,
например `sb.Set(sb.Assign(fieldName, value))`.

## Ошибки и бизнес-исходы

Ошибка из callback отмечает операцию failed в tracing/metrics и может запускать
retry. `pgx.ErrNoRows` и ожидаемую uniqueness condition перехватывайте внутри:
сохраните исход в flag, верните nil, преобразуйте flag в domain error после
`Call`. Сбрасывайте захваченное состояние в начале callback, если он может
повторяться. Технические ошибки оборачивайте с контекстом операции.

## Несколько записей и транзакции

[RunInTx](../pkg/connections/postgres/postgres.go) принимает callback и
`pgx.TxOptions`. Wrapper начинает транзакцию на полученном соединении,
передаёт callback то же соединение как `Queryable`, затем commit/rollback.
Не выполняйте связанные запросы через другой pool/connection внутри callback.

`Call` и `RunInTx` могут повторять **весь callback**, если включён backoff.
Оцените идемпотентность, последствия неоднозначного результата commit и внешние
side effects. Название метода не заменяет чтения wrapper реализации; mutation
и требующие согласованности reads выполняйте на write pool, учитывая replica lag.

Реальный пример исходного STG: `internal/app/repository/todorepo/repository.go`.
Он содержит исторические отличия от новых error conventions из `AGENTS.md`.
Для schema changes см. [Миграции](migrations.md); для проверки —
[Тестирование](integration-testing.md).
