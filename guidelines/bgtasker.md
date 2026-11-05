# BGTasker

`stg/pkg/bgtasker` выполняет короткие ограниченные задачи в worker pool.
Для постоянных pollers и consumers используйте [Grace](grace.md).

## Конфигурация

Подключите `bgtasker.Module` и добавьте секцию **`background`**:

```yaml
background:
  tasks:
    analytic:
      runner_type: drop
      async_workers_num: 8
      async_buffer_size: 1024
      async_worker_timeout: 2s
```

Корневой config provider читает `background.tasks` и нормализует настройки.
Имена задач выбирает приложение. `pool.Get(name)` получает runner; вызывайте
его при построении Fx-графа, до запуска lifecycle. Неизвестное имя до старта
создаёт runner с defaults и warning, после старта — ошибку. Явная секция помогает
избежать случайного использования defaults.

## Consumer-owned интерфейс

Бизнес-сервис объявляет локальный интерфейс:

```go
package analytics

import (
    "context"

    "stg/pkg/bgtasker"
)

type Runner interface {
    Run(context.Context, bgtasker.TaskFunc)
}

type Service struct {
    runner Runner
}

func New(runner Runner) *Service {
    return &Service{runner: runner}
}

func (s *Service) RunTask(ctx context.Context, task bgtasker.TaskFunc) {
    s.runner.Run(ctx, task)
}
```

В `fx.go` того же потребителя получите `*bgtasker.Pool`, вызовите `Get("analytic")`
и верните runner как локальный интерфейс через `fx.Out`. Provider возвращает
ошибку Fx, а не подменяет зависимость no-op реализацией.

## Семантика выполнения

| Режим | Когда очередь заполнена |
|---|---|
| `drop` (default) | Задача отбрасывается; ошибка логируется |
| `block` | Вызывающая горутина ждёт места в очереди |

`Run` не возвращает ошибку или результат задачи. Ошибки выполнения логируются;
повторов и durable queue нет. `block` обеспечивает backpressure, но не
гарантирует сохранность данных при падении процесса. Не используйте это
обещание как основание для фонового проведения платежей.

Runner сохраняет значения context через `context.WithoutCancel`, затем создаёт
собственный timeout для выполнения. Отмена request context не отменяет задачу.
Timeout кооперативный: task должна передавать context операциям и реагировать
на отмену. Он не прерывает произвольный Go-код принудительно.

Не передавайте в очередь изменяемые request buffers, pointers или maps,
которые caller продолжает менять. Подготовьте самостоятельные данные до `Run`.
Внутри task используйте переданный `taskCtx`; повторный `WithoutCancel(taskCtx)`
снимет установленный runner timeout.

При остановке runner закрывает очередь и ждёт workers. Планируйте ограниченное
время задач и согласуйте окончание enqueue с lifecycle приложения.

Источники: [config](../pkg/bgtasker/config.go), [pool](../pkg/bgtasker/pool.go),
[runner](../pkg/bgtasker/runner.go).
