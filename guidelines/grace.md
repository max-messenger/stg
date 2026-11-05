# Работа с Grace

`stg/pkg/grace` управляет lifecycle постоянных сервисов. Интерфейс:

```go
type Service interface {
    Start(context.Context) error
    Stop(context.Context) error
}
```

`Start` context ограничивает старт, а не время жизни сервиса. Loop context
создавайте отдельно и связывайте с отменой в `Stop`. Все child workers получают
этот context и обязаны завершаться до возврата из `Stop`.

## Пример постоянного сервиса

Ниже самостоятельный пример для сервиса, который Fx запускает и останавливает
один раз. `process` — обязательная зависимость, переданная конструктору.

```go
package cleanup

import (
    "context"
    "sync"
    "time"

    "go.uber.org/zap"
)

type Service struct {
    log     *zap.Logger
    process func(context.Context) error
    cancel  context.CancelFunc
    wg      sync.WaitGroup
}

func New(log *zap.Logger, process func(context.Context) error) *Service {
    return &Service{log: log, process: process}
}

func (s *Service) Start(_ context.Context) error {
    ctx, cancel := context.WithCancel(context.Background())
    s.cancel = cancel
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                opCtx, opCancel := context.WithTimeout(ctx, 10*time.Second)
                err := s.process(opCtx)
                opCancel()
                if err != nil && ctx.Err() == nil {
                    s.log.Error("cleanup failed", zap.Error(err))
                }
            }
        }
    }()
    return nil
}

func (s *Service) Stop(_ context.Context) error {
    s.cancel()
    s.wg.Wait()
    return nil
}
```

Операция должна соблюдать context. `WithTimeout` не может принудительно остановить
код, игнорирующий отмену. Если loop создаёт children, учитывайте их в ожидании
и ограничивайте concurrency.

## Регистрация в Fx

В `fx.go` потребителя зарегистрируйте `New`, provider его обязательных
зависимостей и адаптер:

```go
type GraceOut struct {
    fx.Out
    Service grace.Service `group:"grace"`
}

func GraceAdapter(s *Service) GraceOut {
    return GraceOut{Service: s}
}
```

Для этого фрагмента нужны imports `go.uber.org/fx` и `stg/pkg/grace`.
Провайдьте адаптер рядом с конструктором в named package `Module`; модуль
подключите к приложению через его layer aggregator.

## Порядок и ограничения

[ServicePool](../pkg/grace/grace.go) вызывает `Start` последовательно и
`Stop` параллельно; порядок элементов Fx-группы не задаёт dependency ordering.
Не рассчитывайте, что другой сервис уже остановлен, когда начинается ваш `Stop`.
Box связывает pool с Fx lifecycle.

Текущий пример MAX subscriber запускает отдельную горутину на update и не ждёт
workers в `Stop`. Он показывает существующий dispatch, но не является образцом
bounded concurrency или корректного shutdown. Для новой реализации используйте
отменяемый parent context, ограниченные workers и ожидание завершения.

Для коротких задач с очередью используйте [BGTasker](bgtasker.md).
После lifecycle changes запускайте focused tests с `-race`.
