[![Apache 2.0](https://img.shields.io/badge/License-Apache%20License%202.0-blue.svg)](LICENSE)

# Фреймворк для настройки ботов

- [Фреймворк для настройки ботов](#фреймворк-для-настройки-ботов)
    - [Описание](#описание)
    - [Установка фреймворка](#установка-фреймворка)
    - [Пример быстрого старта](#пример-быстрого-старта)
    - [Настройка работы бота](#настройка-работы-бота)
        - [Подключение подписок на события](#подключение-подписок-на-события)
        - [Обработка событий](#обработка-событий)
        - [Обработка команд](#обработка-команд)
        - [Обработка сообщений](#обработка-сообщений)
        - [Обработка Callback-запросов](#обработка-callback-запросов)
        - [Объект `Context`](#объект-context)
        - [Промежуточный обработчик (Middleware)](#промежуточный-обработчик-middleware)
            - [Обработка ошибок в Middleware](#обработка-ошибок-в-middleware)
    - [Пример](#пример)
    - [Как предложить улучшение или идею](#как-предложить-улучшение-или-идею)
    - [Лицензия](#лицензия)

## Описание

Это Golang-фреймворк для настройки работы [ботов для MAX](https://dev.max.ru/docs). С помощью него вы можете настраивать бота, обрабатывать сообщения, команды, callback-запросы и события

* **Обработка команд** — маршрутизация для событий бота
* **Контекстная обработка** — каждое обновление обрабатывается с контекстом, который содержит данные, специфичные для запроса
* **Поддержка Middleware** — создание промежуточных обработчиков для логирования, аутентификации, ограничения частоты запросов и других задач
* **Модульная архитектура** — бот расширяется и настраивается под запросы

## Установка фреймворка

Для установки используйте команду `go get`:

```go
go get github.com/max-messenger/maxbot
```

## Пример быстрого старта

1. [Создайте бота](https://dev.max.ru/docs/chatbots/bots-create/create) на платформе MAX для партнёров
2. [Получите токен бота (access_token)](https://dev.max.ru/docs/chatbots/bots-create/manage#%D0%9F%D0%BE%D0%BB%D1%83%D1%87%D0%B5%D0%BD%D0%B8%D0%B5%20%D1%82%D0%BE%D0%BA%D0%B5%D0%BD%D0%B0%20%D0%B1%D0%BE%D1%82%D0%B0) и установите его как переменную окружения `BOT_TOKEN`
3. Создайте файл `main.go` и вставьте следующий код:

```go
package main

import (
	"log"
	"os"

	"github.com/max-messenger/maxbot"
)

func main() {
	// Получение токена бота из переменных окружения
	access_token:= os.Getenv("BOT_TOKEN")

	// Создание нового экземпляра бота
	bot, err := maxbot.NewApi(access_token)
	if err != nil {
		log.Fatal(err)
	}

	// Определение обработчика для команды /start
	bot.Handle("/start", func(ctx *maxbot.Context) error {
		return ctx.Reply("Привет! Я MaxBot. Чем я могу помочь?")
	})

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
```

4. Для запуска бота в терминале выполните команду:

```go
go run main.go
```

После выполнения команды в терминале отобразится сообщение: `Бот запускается...`

5. Откройте диалог с вашим ботом в MAX и отправьте команду `/start`. Бот ответит вам приветственным
   сообщением:
   `Привет! Я MaxBot. Чем я могу помочь?`

Готово! Ваш бот работает

## Настройка работы бота

### Подключение подписок на события

Чтобы настроить бота на получение обновлений через [Webhook - подписку](https://dev.max.ru/docs-api#Webhook), укажите `access_token`, URL и список типов событий для подписки:

```go 
bot, err := maxbot.NewApi(access_token, maxbot.WithWebhook("https://your-domain.com/webhook"))
// Типы событий: создание сообщений, получение callback
maxbot.OnMessageCreated,
maxbot.OnMessageCallback,
```

### Обработка событий

Для регистрации обработчика необходимых событий используйте метод `Handle`:

```go
// Обработка события изменения названия чата
bot.Handle(maxbot.OnChatTitleChangedEvent, func (c maxbot.Context) error {
    return c.Send("Заголовок чата изменён")
})

```

### Обработка команд

Для регистрации обработчика необходимых команд используйте метод `Handle`:

```go
// Обработка команды
bot.Handle("/start", startHandler)
bot.Handle("/help", helpHandler)
```

### Обработка сообщений

Для регистрации обработчика необходимых сообщений используйте метод `Handle`:

```go
// Обработка всех текстовых сообщений (не начинающихся с '/')
bot.Handle(maxbot.OnMessageCreated, textHandler)
```

### Обработка Callback-запросов

Для регистрации обработчика callback-запросов используйте метод `Handle`:

```go
// Связь callback-данных "pushBtn" с обработчиком `Context`
bot.HandleCallback("pushBtn", func (c maxbot.Context))
```

### Объект `Context`

Объект `maxbot.Context` передаётся в каждый обработчик и содержит всю информацию о сообщении, а также методы для ответа

```go
func myHandler(ctx *maxbot.Context) error {
    // Получение текстового сообщения
    text := ctx.Message.Text
    
    // Ответ сообщением
    return ctx.Reply("Вы сказали: " + text)
}
```

### Промежуточный обработчик (Middleware)

Промежуточные обработчики (Middleware) позволяют выполнять код до или после основных обработчиков. Используются для логирования, аутентификации или сбора метрик

```go
// Определение Middleware для логирования
loggingMiddleware := func (next maxbot.HandlerFunc) maxbot.HandlerFunc {
    return func (ctx *maxbot.Context) error {
        log.Printf("Получено обновление от пользователя %d", ctx.Message.From.ID)
        return next(ctx)
	}
}

// Применение Middleware к конкретному обработчику
bot.Handle("/secret", secretHandler, loggingMiddleware)

// Глобальное применение Middleware
bot.Use(loggingMiddleware)
```

#### Обработка ошибок в Middleware

Чтобы перехватить и классифицировать ошибки с помощью Middleware, используйте код:

```go
errorMiddleware := func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
    return func(ctx *maxbot.Context) error {
        err := next(ctx)
        if err != nil {
            if errors.Is(err, ErrUserNotFound) {
                return ctx.Reply("Пользователь не найден")
            }
            return ctx.Reply("Внутренняя ошибка")
        }
        return nil
    }
}

bot.Use(errorMiddleware)
```

## Пример

Пример [базового бота, демонстрирующего обработку команд и ответы](example/simple/README.md)

## Как предложить улучшение или идею

Проект имеет открытый исходный код — вы можете сделать в проект `Pull Request` со своими доработками:

1. Сделайте `Fork` репозитория
2. Создайте ветку для вашей функции: `git checkout -b feature/amazing-feature`
3. Зафиксируйте изменения: `git commit -m 'Add some amazing feature`'
4. Отправьте изменения в ветку: `git push origin feature/amazing-feature`
5. Откройте `Pull Request`

## Лицензия

Этот проект лицензирован под Apache License 2.0 – подробности смотрите в файле [LICENSE](LICENSE)
