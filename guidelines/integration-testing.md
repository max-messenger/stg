# Тестирование

Команды ниже выполняются из корня исходного STG. Источник истины —
[Makefile](../Makefile).
`create.sh` удаляет `tests/`, `testkit/` и CI; в новом приложении создайте
собственные suites и проверьте доступные команды.

## Имена и изоляция

- Unit: `TestUnit...`, моки, без внешних БД/HTTP/инфраструктуры.
- Integration: `TestIntegration...`, реальные хранилища.
- E2E: `TestE2E...`, HTTP API работающего приложения.

Новые unit-тесты пишите в external `<package>_test` с `gomock.NewController(t)`
и `testify/require`. Интерфейс и mockgen directive принадлежат потребителю.
Текущие integration suites используют собственные package names и harnesses;
для расширения сначала изучите конкретный suite.

## Локальные команды

| Команда | Что реально выполняет |
|---|---|
| `make test_unit` | gotestsum, `-race`, coverage, **все** тесты `./internal/...`, без `-run` |
| `make test_integration` | gotestsum, `-race`, coverage, `-run TestIntegration ./...` |
| `make test_e2e` | gotestsum, `-run TestE2E ./...`, без coverage/race |
| `make test_cover` | Повторный unit run и summary из `coverage/unit.txt` |
| `make generate` | `go generate -x ./...` — mocks и Swagger directives |
| `golangci-lint run` | Read-only lint |
| `make lint` | Lint с `--fix`, может изменить файлы |

В `make test_unit` не входят `pkg/` и `testkit/`. Для соответствующих изменений
запускайте focused package tests, например `go test -race ./pkg/grace/...`.
Не размещайте infrastructure tests в `internal/...`: unit target запускает
там все тесты независимо от имени.

`-run` фильтрует тесты, но не предотвращает выполнение `TestMain` другого
пакета. Integration/E2E Make targets обходят `./...` и могут инициализировать
другие suites. Для узкой проверки выбирайте точный package.

## Integration suites

Текущие suites находятся в `tests/integrations_todorepo`,
`tests/integrations_staterepo`, `tests/integrations_tokenrepo` и
`tests/integrations_filerepo`. Они используют `testosteron` для запуска Docker
контейнеров из `TestMain`, настройки env и cleanup. Это не описанный ранее
`sync.Once` / `ensureSetup` harness рядом с репозиторием.

Например, `tests/integrations_todorepo/main_test.go` поднимает
PostgreSQL, выставляет DSN, применяет миграции и создаёт repository.
Пути и env names определены в `tests/const.go`, общий config —
`config/stg/config.runner.yaml`. Нужны Docker и доступ к образам suites.
Предварительный `make dev_env_up` не заменяет их контейнерный lifecycle.

## E2E

Тесты в `tests/e2e` отправляют HTTP-запросы; harness приложения не запускает.
Перед запуском подготовьте инфраструктуру и running app. Для ручной локальной
инфраструктуры существует `make dev_env_up`; этот target сначала выполняет
`dev_env_down`, который удаляет volumes через `docker-compose down -v`.

Адрес E2E задаёт `INTEGRATION_STG_HTTP_ADDR` (default `localhost:8080`).
В runner config HTTP bind по умолчанию `:18080`, поэтому при использовании
этого config согласуйте E2E address. Ему также нужен MAX API stub; Compose
такой stub не содержит. Нельзя считать `dev_env_up` готовым E2E окружением.

Для параллельных E2E создавайте изолированные данные. Текущий helper использует
`crypto/rand` и положительный диапазон int32; это снижает коллизии, но не
гарантирует уникальность. Не очищайте общие таблицы в параллельных suites.

**Известное расхождение:** auth harness использует CookieJar, а сервер выдаёт
JSON Bearer-токены. `/me` тоже имеет другую форму ответа. Auth E2E требуется
адаптировать перед использованием как quality gate; см. [Авторизация](auth.md).
`INTEGRATION_STG_SHORT_ACCESS_TTL=1` включает TTL-тесты, но не меняет server TTL:
они дополнительно требуют `access_ttl=3s` и работающего auth harness.

## Coverage

Unit profile: `coverage/unit.txt`; integration profile при локальном target:
`coverage/integration.txt`. `make test_cover` **не объединяет** profiles.
При самостоятельном объединении profiles их modes должны совпадать.

`.covignore` перечисляет строки, которые исключаются из **unit** profile при
подсчёте coverage.

Факт наличия suites не подтверждает успешный прогон текущего
checkout. В отчёте указывайте точные команды, scope и недоступную инфраструктуру.
