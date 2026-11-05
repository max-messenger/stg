# Чек-лист разработки сервиса

Общие правила — в [AGENTS.md](../AGENTS.md), устройство слоёв — в
[Архитектуре](architecture.md). Примеры исходного STG могут отсутствовать после
`create.sh`; сначала проверьте дерево сгенерированного приложения.

## Планирование

- [ ] Определить entry point, owning service, входы, результат и бизнес-ошибки.
- [ ] Проверить существующие capabilities; добавлять только необходимые слои.
- [ ] Оценить изменения публичного контракта, конфигурации и схемы.
- [ ] Выбрать проверки по текущим Makefile и CI.

## Domain и сервис

- [ ] Общие модели и ожидаемые ошибки разместить в `internal/app/domain`.
- [ ] Создать focused service package с методами по ответственности.
- [ ] Интерфейсы зависимостей объявить в потребителе, добавить mockgen directive.
- [ ] Экспортировать package `Module`, конструктор и адаптеры в его `fx.go`.
- [ ] Адаптеры получают providers через `fx.In`, возвращают локальные интерфейсы
  через `fx.Out` и содержат присваивания полей.
- [ ] Обязательные Fx-зависимости не подменять defaults/no-op; defaults конфигурации
  задавать в provider.
- [ ] Зарегистрировать новый модуль в соответствующем layer aggregator.

## Persistence и внешние системы

- [ ] Для нового repository определить один owning service.
- [ ] Использовать локальные PostgreSQL/Redis wrappers, стабильные operation names.
- [ ] SQL строить параметризованно с `go-sqlbuilder`, определить table/column constants.
- [ ] Reads направлять в read pool; mutations — в write pool.
- [ ] Проверить atomicity и retries для нескольких записей.
- [ ] Бизнес-исходы вынести за пределы `Call` callbacks; технические ошибки оборачивать.
- [ ] Закрывать rows/readers, проверять `rows.Err()`.
- [ ] Redis keys строить через formatter; structured values кодировать marshaler.
- [ ] TTL задавать для данных с ограниченным временем жизни.
- [ ] Для schema change создать новую миграцию; оценить совместимость и recovery.
- [ ] Внешнюю систему оформить client package; сервис потребляет локальный интерфейс.

Подробности: [PostgreSQL](postgresql.md), [Redis](redis.md), [S3](s3.md),
[Kafka](../pkg/connections/kafka/README.md), [Миграции](migrations.md).

## HTTP

- [ ] Контроллер создать в `internal/app/router/<feature>ctrl` с локальным service interface.
- [ ] Routes зарегистрировать в `Register`; controller экспортировать в группу `controller`.
- [ ] Провалидировать path/query/body до сервиса и передать `r.Context()`.
- [ ] Использовать `stg/pkg/marshaler` и общие router response helpers.
- [ ] Domain errors проверять через `errors.Is`, технические детали клиенту не раскрывать.
- [ ] Добавить Swagger annotations к бизнес-handler и регенерировать artifacts.
- [ ] Подключить controller module на уровне приложения.

## Фоновая работа

- [ ] Для bounded tasks выбрать BGTasker и секцию `background.tasks`.
- [ ] Получить runner до старта Fx lifecycle и адаптировать к локальному интерфейсу.
- [ ] Выбрать `block` для backpressure или `drop` для намеренно lossy работы;
  не считать worker queue durable storage.
- [ ] Не передавать изменяемые request data; соблюдать timeout task context.
- [ ] Постоянный сервис оформить как `grace.Service` в группе `grace`.
- [ ] `Start` запускает loop без привязки lifetime к startup context.
- [ ] `Stop` отменяет loop/children и ждёт все созданные горутины.
- [ ] Ограничить concurrency; MAX updates читает только subscriber.

Подробности: [BGTasker](bgtasker.md), [Grace](grace.md).

## Проверка

- [ ] Unit-тесты изолированы моками; новые имена начинаются с `TestUnit`.
- [ ] Проверить успех, ожидаемые отказы и важные boundary cases.
- [ ] После изменения интерфейсов/Swagger выполнить `make generate` и проверить diff.
- [ ] После application changes выполнить `make test_unit`.
- [ ] Для framework/test helpers отдельно запустить focused race tests.
- [ ] По необходимости проверить integration/E2E с нужной инфраструктурой.
- [ ] Перед commit/MR выполнить read-only `golangci-lint run`.
- [ ] Проверить документацию, ссылки и совместимость с `create.sh`.

Матрица и ограничения: [Тестирование](integration-testing.md).
