# Тестирование бота

Настоящий документ определяет правила и подход к юнит-тестированию на примере `internal/app/services/bot`.

`create.sh` удаляет этот пример и `testkit/`. В новом приложении сначала
создайте свои bot-компоненты и test helpers; указанные ниже пути относятся
к исходному STG. Общие правила — в [AGENTS.md](../AGENTS.md).

## Назначение тестов

Основной объект тестирования — метод `ProcessUpdate(...)`.
Ключевая проверка — корректность исходящих действий бота:

- `SendMessage`
- `EditMessage`
- `DeleteMessage`

## Используемые инструменты

Для написания тестов применяются:

- `testkit/bot`:
  - `UpdateBuilder` — формирование входных update (`MessageCreated`, `BotStarted`, `MessageCallback`);
  - `ClientSpy` — фиксация исходящих вызовов клиента;
  - `RequireSentCount`, `RequireLastSentText` — базовые assert-хелперы.
- `mockgen` для генерации моков внешних зависимостей:
  - `Store`, `Repository`.

## Рекомендуемый сценарий теста

1. Подготовить зависимости через `prepare(...)`:
   - `ClientSpy`;
   - сгенерированные mockgen-моки.
2. Настроить ожидания для `repo/store`.
3. Сформировать входной update с помощью `UpdateBuilder`.
4. Вызвать `ProcessUpdate(...)`.
5. Проверить outbound-результат (`SendMessage`/`EditMessage`/`DeleteMessage`).

## Пример

```go
func TestUnitProcessUpdate_MessageCreated_HelpCommand(t *testing.T) {
    d := prepare(t)

    upd := testbot.NewUpdateBuilder().
        WithUserID(42).
        WithChatID(77).
        WithText("/help").
        MessageCreated()

    d.service.ProcessUpdate(d.ctx, upd)

    testbot.RequireSentCount(t, d.client, 1)
    testbot.RequireLastSentText(t, d.client, infoMessage)
}
```

Актуальные сценарии см. в `internal/app/services/bot/service_test.go`.
Фрагмент использует `prepare` и `infoMessage`
из этого test file; это пример внутри существующего suite, не отдельный package.

## Генерация моков через mockgen

Базовая генерация для bot-сервиса:

```bash
go generate ./internal/app/services/bot
```

Альтернативный запуск:

```bash
go tool mockgen -source=internal/app/services/bot/service.go -destination=internal/app/services/bot/mocks/service_mocks.go -package=mocks
```

По умолчанию моки bot-сервиса сохраняются в `internal/app/services/bot/mocks`.

## Область применения mockgen

Генерация моков не ограничивается только `internal/app/services/bot`.
При наличии интерфейсов и тестовой необходимости моки могут быть сгенерированы в любом пакете проекта, где это улучшает изоляцию тестов и упрощает проверку поведения.

## Проверка

`make test_unit` запускает все тесты `internal/...` с race detector и coverage,
без name filter. При изменении helpers дополнительно выполните
`go test -race ./testkit/bot/...`. См. [Тестирование](integration-testing.md).
