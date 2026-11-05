# Авторизация

Текущий пример приложения валидирует MAX `initData` и выдаёт пару JWT.
**Токены возвращаются в JSON и передаются через заголовок Authorization;
HttpOnly-cookies сервер не устанавливает.**

Источники контракта: [контроллер](../internal/app/router/authctrl/authctrl.go),
[сервис](../internal/app/services/auth/auth.go) и
[token repository](../internal/app/repository/tokenrepo/repository.go).
Не переносите пример в новый сервис без проверки требований к его сессиям.

## HTTP API

| Метод и путь | Вход | Успешный ответ |
|---|---|---|
| `POST /v1/auth/login` | JSON `{"init_data":"..."}` | `{"data":{"access_token":"...","refresh_token":"..."}}` |
| `POST /v1/auth/refresh` | `Authorization: Bearer <refresh_token>` | Такая же пара токенов в `data` |
| `GET /v1/auth/me` | `Authorization: Bearer <access_token>` | `{"user_id":123,"username":"...","chat_id":456}` без `data` envelope |
| `POST /v1/auth/logout` | Текущая реализация принимает raw refresh JWT в `Authorization` | `{}` |

Login возвращает 400 для пустого/некорректного тела и 401 при отказе сервиса.
Refresh и logout возвращают 401 при отказе сервиса. `/me` при ошибке возвращает
401 с текстом, а не JSON envelope. Это текущие особенности примера;
для новых handlers используйте response/error conventions из `AGENTS.md`.

Logout пока не удаляет префикс `Bearer `, в отличие от refresh и `/me`.
В Swagger login также остался путь `/v1/auth/token`, хотя реально зарегистрирован
`/v1/auth/login`. При изменении этих особенностей обновите handlers, annotations,
сгенерированный Swagger и тесты совместно.

## initData и токены

Сервис делегирует проверку `initData` MAX SDK (`maxbot.ValidateInitData`),
используя `auth.max_token`. Не дублируйте алгоритм SDK в бизнес-коде. Отдельной
настройки `auth_ttl` в текущем `auth.Config` нет.

Access и refresh подписываются HS256. Текущие claims: `user_id`, `chat_id`,
`user_name`, `token_type`, `await` и `expire`. `expire` — custom claim,
проверяемый сервисом, а не стандартный `exp`. Полного профиля пользователя
или `expires_in` в текущих HTTP-ответах нет.

Redis хранит `domain.AuthData` по ключу `refresh:<userID>` с pool prefix и TTL.
Отдельного tokenID в ключе нет: повторный login заменяет сессию пользователя.
Refresh проверяет сохранённый JWT, удаляет старую запись и сохраняет новую пару.
Эти шаги не атомарны; документация не обещает защиту от одновременного replay.
JWT формируются с секундной точностью без случайного идентификатора, поэтому
повторная выдача в ту же секунду не гарантирует отличающуюся строку токена.

## Middleware

[AuthMiddleware](../pkg/http/middlewares/auth.go) читает `Authorization`,
вызывает переданную функцию проверки и сохраняет userID в context.
Для получения ID используется `middlewares.GetUserID(ctx)`.
`CookieAuth` и `UserIDFromCtx` в текущем framework отсутствуют.

Todo-контроллер защищён этим middleware. В его текущем API userID также приходит
в теле запроса; сервис сравнивает его с ID из context через `ValidateUserID`.

## Конфигурация

```yaml
auth:
  jwt_secret: ${JWT_SECRET}
  max_token: ${MAXBOT_TOKEN}
  access_ttl: ${ACCESS_TTL:"15m"}
  refresh_ttl: ${REFRESH_TTL:"720h"}
```

Эти поля читает [auth.Config](../internal/app/services/auth/config.go).
Поля `cookie_secure`, `cookie_same_site`, `cookie_path`, `auth_ttl` не поддерживаются.
TTL repository получает через свой provider из auth-секции.

[CORS middleware](../pkg/http/middlewares/cors.go) при пустом `allowed_origins`
отражает любой Origin и выставляет `Allow-Credentials`. Это разрешение всех
origins, а не более безопасная альтернатива allowlist. Для ограниченного
доступа задавайте явные допустимые origins.

## Тестирование

Unit-тесты сервиса находятся в `internal/app/services/auth/auth_test.go`.
Основная команда — `make test_unit`.

Существующие `tests/e2e/auth_test.go` и их harness всё ещё ожидают cookies и
старую форму `/me`. Они требуют адаптации к текущему Bearer-контракту; наличие
этих тестов не означает, что текущая авторизация проверена E2E.
Флаг `INTEGRATION_STG_SHORT_ACCESS_TTL` лишь разрешает TTL-тесты, но не меняет
конфигурацию сервера. Подробнее: [Тестирование](integration-testing.md).
