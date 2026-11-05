# Queue

Общая очередь на Redis Streams: одно подключение к Redis, настройки по именам
очередей, произвольный `[]byte` payload и callback для обработки. Рейтлимиты,
сериализация и бизнес-логика остаются в callback. Требуется Redis 6.2+.

## Подключение через Fx

Добавить модули в приложение:

```go
import (
    "go.uber.org/fx"

    "stg/pkg/connections/rediscli"
    "stg/pkg/queue"
)

fx.Options(
    rediscli.Module,
    queue.Module,
)
```

В конфиге выбрать одно подключение для всей очереди:

```yaml
redis:
  main:
    addrs: ["127.0.0.1:6379"]
    key_prefix: my_service

queue:
  redis:
    pool_name: main
    prefix: queues
  defaults:
    group: workers
    concurrency: 1
    ack_mode: on_success
  queues:
    notifications:
      concurrency: 4
    webhooks:
      concurrency: 10
      ack_mode: before_handle
```

После этого `*queue.Queue` доступен через DI. Имя очереди передаётся при вызове,
настройки выбираются автоматически:

```go
func NewService(q *queue.Queue) *Service {
    return &Service{queue: q}
}

func (s *Service) Publish(ctx context.Context, payload []byte) error {
    return s.queue.Send(ctx, "notifications", payload)
}

func (s *Service) Run(ctx context.Context) error {
    // Использует concurrency: 4 и ack_mode: on_success.
    return s.queue.Consume(ctx, "notifications",
        func(ctx context.Context, msg queue.Message) error {
            // Рейтлимиты и обработка реализуются здесь.
            return s.handle(ctx, msg.Payload)
        },
    )
}
```

`Consume` блокирует до отмены контекста. Вызывать его в фоновой задаче с контекстом
жизненного цикла сервиса. Модуль сам не запускает обработчики.

## Consumer как фоновый сервис STG

Постоянный consumer регистрируется в группе `grace`. Контекст `Start` ограничивает
запуск приложения, поэтому для чтения создаётся отдельный контекст, отменяемый в
`Stop`. Пример для сервиса, запускаемого и останавливаемого Fx один раз:

```go
package notifications

import (
    "context"

    "go.uber.org/fx"
    "go.uber.org/zap"

    "stg/pkg/grace"
    "stg/pkg/queue"
)

type Consumer struct {
    log     *zap.Logger
    queue   *queue.Queue
    process queue.ConsumerCallback
    cancel  context.CancelFunc
    done    chan struct{}
}

func New(log *zap.Logger, q *queue.Queue, process queue.ConsumerCallback) *Consumer {
    return &Consumer{log: log, queue: q, process: process}
}

func (c *Consumer) Start(_ context.Context) error {
    ctx, cancel := context.WithCancel(context.Background())
    c.cancel = cancel
    c.done = make(chan struct{})
    go func() {
        defer close(c.done)
        if err := c.queue.Consume(ctx, "notifications", c.process); err != nil && ctx.Err() == nil {
            c.log.Error("queue consumer stopped", zap.Error(err))
        }
    }()
    return nil
}

func (c *Consumer) Stop(ctx context.Context) error {
    c.cancel()
    select {
    case <-c.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

type GraceOut struct {
    fx.Out
    Service grace.Service `group:"grace"`
}

func graceAdapter(c *Consumer) GraceOut { return GraceOut{Service: c} }

var Module = fx.Module("notifications", fx.Provide(New, graceAdapter))
```

В приложении подключите `notifications.Module` рядом с `queue.Module` и
предоставьте обязательный `queue.ConsumerCallback` через `fx.Provide` — например,
провайдер возвращает метод обработки бизнес-сервиса. `grace.Module` уже включён в
стандартные модули STG. Бизнес-обработчик должен соблюдать отмену контекста:
`Consume` ждёт завершения активных callback, а дедлайн `Stop` ограничивает только
ожидание. При ошибке создания Redis-группы `Consume` завершается; в этом примере
ошибка логируется, автоматического перезапуска consumer нет.

Подробнее о lifecycle и регистрации сервисов: [Работа с Grace](../../guidelines/grace.md).

## Без Fx

Можно передать существующий `*rediscli.Redis`:

```go
q, err := queue.New(queue.Config{
    Redis: queue.RedisConfig{Prefix: "queues"},
    Defaults: queue.Settings{
        Group:   "workers",
        AckMode: queue.AckOnSuccess,
    },
    Queues: map[string]queue.Settings{
        "notifications": {Concurrency: 4},
    },
}, logger, rds)
```

`Queue` не закрывает Redis-клиент. В Fx подключением управляет `rediscli.Module`,
без Fx — приложение.

В standalone-конструктор можно передать `nil` вместо логгера: используется
`zap.NewNop()`. Через Fx логгер — обязательная зависимость.

## API

- `Send(ctx, stream, payload) error` — публикация.
- `Publish(ctx, stream, payload) (string, error)` — публикация с возвратом Redis ID.
- `Consume(ctx, stream, callback) error` — чтение и обработка; при остановке
  возвращает `ctx.Err()` и ждёт завершения активных callback.

Сообщение содержит `ID`, логическое имя очереди `Stream`, `Payload` и `Timestamp`,
полученный из Redis ID. Несколько вызовов `Consume` для одной очереди и одной
группы делят сообщения между собой. Один объект `Queue` можно использовать
конкурентно для нескольких streams. Лимит `concurrency` действует на каждый
вызов `Consume` отдельно. При параллельной обработке порядок завершения не гарантируется.

## Подтверждение и ошибки

| `ack_mode` | Поведение |
| --- | --- |
| `on_success` (по умолчанию) | ACK после `nil` от callback. Ошибка оставляет сообщение pending для повторной доставки. |
| `before_handle` | ACK до callback. После подтверждения сообщение может потеряться при падении процесса. |
| `after_handle` | ACK после callback, включая ошибку или panic. При падении процесса до ACK возможна повторная доставка. |

При отмене или таймауте контекста обработки поздний ACK не выполняется. В режиме
`before_handle` подтверждение к этому моменту уже выполнено.

Pending-сообщения восстанавливаются через `XAUTOCLAIM`. Пока callback активен,
очередь обновляет его владение каждые `reclaim_idle / 3`, чтобы длительная обработка
или ожидание rate limit не приводили к обычной повторной доставке. При потере
владения или ошибке обновления контекст callback отменяется. Повторы всё равно
возможны при сбоях, поэтому для `on_success` нужен идемпотентный обработчик.

Бизнес-повторы внутри callback реализует приложение. Возвращённая ошибка в режиме
`on_success` приводит к повторной доставке без ограничения числа попыток после
истечения `reclaim_idle`. Для своей DLQ можно опубликовать сообщение в другой stream
и вернуть `nil`; эти два действия не атомарны.

Ошибки транспорта и callback логируются через переданный конструктору логгер.
Ошибки callback не завершают `Consume`, ошибки чтения повторяются с паузой 500ms.
Panic callback перехватывается и считается ошибкой. Callback должен учитывать
`ctx.Done()`: таймаут не может принудительно остановить Go-функцию.

## Настройки

Конфиг разделён на `redis`, `defaults` и `queues`. В `redis` задаются:

- `pool_name` — имя подключения в `rediscli.Pool`, обязательно для Fx;
- `prefix` — общий префикс Redis streams, по умолчанию пустой.

В примере выше физический ключ для `notifications` —
`my_service:queues:notifications`: сначала prefix Redis-подключения, затем prefix
очереди и её имя. В сообщении и метриках остаётся логическое имя `notifications`.

Поля `defaults` и каждого элемента `queues` соответствуют `queue.Settings`:

| Поле | По умолчанию | Назначение |
| --- | --- | --- |
| `group` | `workers` | Группа потребителей. |
| `ack_mode` | `on_success` | Момент подтверждения. |
| `concurrency` | `1` | Число параллельных callback на вызов `Consume`. |
| `batch_size` | `concurrency` | Максимальная пачка чтения; ограничивается свободными слотами обработки. |
| `block_timeout` | `1s` | Время блокирующего чтения. |
| `handler_timeout` | `0` | Таймаут контекста callback; `0` — без отдельного таймаута. |
| `reclaim_interval` | `10s` | Интервал сканирования pending. |
| `reclaim_idle` | `1m` | Сколько ждать без обновления владения до повторной доставки. |
| `start_id` | `0` | Начальная позиция **новой** группы: `0` — весь stream, `$` — только новые записи. На существующую группу не влияет. |

Ненулевые поля именованной очереди переопределяют `defaults`; нулевые и
неуказанные поля наследуются. Затем применяются встроенные значения из таблицы.
Если имя отсутствует в `queues`, используются общие defaults. Например,
`handler_timeout: 0` в именованной очереди наследует общий таймаут, если он задан.
Если `batch_size` нигде не задан, он равен concurrency выбранной очереди.

Настройки проверяются при создании объекта. Конструктор копирует map очередей,
поэтому последующая правка исходного конфига не изменяет работающий `Queue`.

ACK не удаляет записи из stream. Retention настраивается приложением; удаление
необработанных записей через `MAXLEN`/`XTRIM` может привести к потере сообщений.
Очередь сама stream не обрезает.

## Метрики

Счётчики и гистограммы обновляются непосредственно при публикации, обработке,
восстановлении сообщений и ошибках транспорта. Регистрируются в стандартном
Prometheus registry с префиксом `redis_queue_`:

- `published_total`, `processed_total`, `errors_total`, `reclaimed_total`;
- `handler_duration_seconds`, `message_lag_seconds`.

Labels: логическое имя stream, группа, ошибка или операция в зависимости от метрики.
Label `stream` всегда содержит имя очереди, переданное в `Send`, `Publish` или
`Consume`, независимо от наличия настроек в `queue.queues`.
Использовать ограниченный набор стабильных имён очередей; не включать в имена
request ID, user ID или другие динамические значения, чтобы не создавать
неограниченное число серий метрик.

Пустое чтение (`redis.Nil`) и существующая группа (`BUSYGROUP`) считаются успешными
результатами внутри Redis-обёртки и не отмечаются как ошибки в её метриках и spans.

Для Redis Cluster все операции выполняются по одному stream-ключу: разные streams
не обязаны иметь общий hash tag.

## Трейсинг

Очередь использует `otel.Tracer("queue")` и общий OTel provider/propagator приложения.
В STG их настраивает `telemetry.Module`. При standalone-использовании настройка
OpenTelemetry выполняется приложением.

- `queue.publish` — span типа `Producer`, с именем очереди, размером payload и Redis ID.
- `queue.process` — span типа `Consumer` на каждую попытку обработки, включая ACK.
  Содержит очередь, группу, consumer, ID сообщения и режим подтверждения.

При публикации trace-контекст добавляется в поля Redis-записи через общий propagator
(`traceparent`/`tracestate` при стандартной настройке STG). Consumer извлекает его
и передаёт контекст span обработки в callback. Вызовы из callback продолжают trace
публикации, в том числе при обработке на другом инстансе. Payload не меняется и
не записывается в атрибуты span.

Ошибки публикации, обработки, ACK и обновления владения записываются в span
состоянием `Error`; panic callback тоже считается ошибкой. Отмена контекста при
штатной остановке не помечает span ошибкой.

Вызовы Redis, включая чтение, reclaim, ACK и heartbeat, дополнительно трассируются
существующим `rediscli.Call`.

## Проверки

```bash
go test -race -run Unit ./pkg/queue
docker compose up -d redis
REDIS_SEED=127.0.0.1:6379 go test -race ./pkg/queue
golangci-lint run ./pkg/queue/...
```

Команды выполняются из корня STG. Без `REDIS_SEED` интеграционные тесты
пропускаются. `make test_unit` запускает только `internal/...`, поэтому для
проверки очереди нужен отдельный вызов `go test` из примера выше.
