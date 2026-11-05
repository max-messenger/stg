# Миграции базы данных

Управление изменениями схемы базы данных с помощью `sql-migrate`.

## Концепция

STG использует библиотеку `rubenv/sql-migrate` для управления миграциями PostgreSQL. Миграции запускаются автоматически при старте приложения через FX lifecycle hook.

Миграции позволяют:

- Отслеживать изменения схемы БД в виде кода
- Применять изменения последовательно

## Структура файлов миграций

Миграции хранятся в папке `db/migrations`.

### Нейминг миграций

Формат имени: `{timestamp}-{description}.sql`

```bash
20260122182108-create_users_table.sql
```

- **timestamp** - `YYYYMMDDHHMMSS` (формат sql-migrate)
- **description** - краткое описание изменения

## Конфигурация

```yaml
migrate:
  pool_name: "stg-upd"  # имя пула для миграций (обычно write pool)
```

Миграции подключаются к приложению через FX:

Подключение `migrate.Module` обязательно; после `create.sh` проверьте и добавьте
его в модули нового приложения при необходимости. Миграции читаются из embed
filesystem `db.Migrations`, а не с диска работающего контейнера.

```go
// internal/app/fx.go
var Modules = fx.Options(
    // ...
    migrate.Module,  // автоматический запуск миграций при старте
    // ...
)
```

## Создание новой миграции

### Через Makefile

```bash
make migrate-new name=create_users_table
```

### Через sql-migrate напрямую

```bash
go tool sql-migrate new -config=db/conf.yml -env=local create_users_table
```

В результате будет создан файл:

```sql
-- db/migrations/20260319142940-create_users_table.sql
```

## Рекомендации по написанию миграций

### 1. Использование Up/Down

Применение миграций располагается под строчкой ```-- +migrate Up```

Откат миграций располагается под строчкой ```-- +migrate Down```

Пример:
```sql
-- +migrate Up

CREATE TABLE IF NOT EXISTS users
(
    id         serial primary key,
    user_id    int
);

-- +migrate Down

DROP TABLE IF EXISTS users;
```

### 2. IF NOT EXISTS / IF EXISTS

Выбирайте условные конструкции по смыслу миграции, а не автоматически:

```sql
CREATE TABLE IF NOT EXISTS users (...);
DROP TABLE IF EXISTS users;
```

`IF NOT EXISTS` проверяет наличие имени, но не соответствие существующей
таблицы требуемой схеме. Иногда явный отказ лучше молчаливого пропуска при
schema drift. Не изменяйте уже применённые миграции; добавляйте новые.
Перед rollout оцените locks, совместимость старой/новой версии приложения,
backfill и rollback/recovery. Down может удалять данные и не является
автоматически безопасным откатом.

### 3. Нейминг колонок и таблиц

Используйте snake_case для названий:

```sql
-- Хорошо
CREATE TABLE user_profiles (
    first_name VARCHAR(255),
    last_name VARCHAR(255)
);

-- Плохо
CREATE TABLE UserProfile (
    FirstName VARCHAR(255),
    LastName VARCHAR(255)
);
```

## Дополнительно

- [rubenv/sql-migrate documentation](https://github.com/rubenv/sql-migrate)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)
