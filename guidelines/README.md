# Гайдлайны по разработке сервисов STG

Документация по разработке сервисов с использованием фреймворка STG.

Общие соглашения и матрица обязательных проверок — в [AGENTS.md](../AGENTS.md).
Гайдлайны описывают исходный репозиторий с примерным приложением. `create.sh`
удаляет часть примеров, tests/testkit, deployment и CI; в сгенерированном
приложении сначала проверьте наличие упомянутых компонентов. Импорты и имена
приложения переименовываются, идентификаторы скиллов `stg-*` остаются прежними.

## Основные документы

- [Архитектура приложения](architecture.md) - обзор принципов и архитектуры STG
- [Авторизация](auth.md) - текущий MAX initData/Bearer-контракт и известные ограничения примера
- [Работа с Grace](grace.md) - управление фоновыми сервисами и graceful shutdown
- [PostgreSQL](postgresql.md) - работа с базой данных, read/write реплики, транзакции
- [Redis](redis.md) - работа с Redis
- [Kafka](../pkg/connections/kafka/README.md) - продюсеры, консьюмеры и консьюмер-группы
- [S3 (MinIO)](s3.md) - файловое хранилище, загрузка/скачивание/удаление файлов
- [BGTasker](bgtasker.md) - асинхронное выполнение задач в фоне
- [Миграции БД](migrations.md) - управление изменениями схемы БД
- [Интеграционное тестирование](integration-testing.md) - три уровня тестов: Unit, Integration (репозитории + БД), E2E (HTTP API)
- [Тестирование бота](bot-testing.md) - unit-тесты `ProcessUpdate`, testkit и mockgen
- [Чек-лист разработки](checklist.md) - пошаговый список действий

## Быстрый старт

Для начала разработки нового сервиса рекомендуется следовать порядку:

1. Ознакомиться с [архитектурой](architecture.md)
2. Изучить примеры:
   - [PostgreSQL](postgresql.md) - для работы с БД
   - [BGTasker](bgtasker.md) - для фоновых задач
   - [Redis](redis.md) - для работы с Redis
3. Следовать [чек-листу](checklist.md) при разработке

## Полезные ссылки

- [Uber FX](https://github.com/uber-go/fx) - Dependency Injection
- [pgx PostgreSQL driver](https://github.com/jackc/pgx) - PostgreSQL драйвер
- [go-redis](https://github.com/redis/go-redis) - Redis клиент
- [franz-go](https://github.com/twmb/franz-go) - Kafka клиент
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) - Observability
