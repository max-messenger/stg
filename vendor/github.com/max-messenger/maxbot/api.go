package maxbot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

type conf struct {
	opts       []maxbot.Opt
	webhookURL string
	secret     string
	types      []string
}

type Api struct {
	ctx    context.Context
	cancel context.CancelFunc
	client *maxbot.Api
	conf   conf

	group *Group

	onError  func(error, Context)
	handlers map[string]HandlerFunc

	Info model.BotInfo
}

func NewApi(token string, opt ...Opt) (*Api, error) {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Api{
		ctx:      ctx,
		cancel:   cancel,
		group:    new(Group),
		onError:  defaultOnError,
		handlers: make(map[string]HandlerFunc),
	}

	for _, o := range opt {
		o(b)
	}

	cli, err := maxbot.NewApi(token, b.conf.opts...)
	if err != nil {
		return nil, err
	}

	b.client = cli

	b.Info, err = b.client.Bots.GetMyInfo(b.ctx)
	if err != nil {
		return nil, err
	}

	err = b.webhookSubscribe()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (a *Api) Use(middleware ...MiddlewareFunc) {
	a.group.Use(middleware...)
}

func (a *Api) Handle(endpoint string, h HandlerFunc, m ...MiddlewareFunc) {
	if len(a.group.middleware) > 0 {
		m = appendMiddleware(a.group.middleware, m)
	}
	a.handlers[endpoint] = func(c Context) error {
		return applyMiddleware(h, m...)(c)
	}
}

func (a *Api) HandleCallback(endpoint string, h HandlerFunc, m ...MiddlewareFunc) {
	if len(a.group.middleware) > 0 {
		m = appendMiddleware(a.group.middleware, m)
	}
	a.handlers[callbackPrefix+endpoint] = func(c Context) error {
		return applyMiddleware(h, m...)(c)
	}
}

func (a *Api) Start() {
	if a.conf.webhookURL != "" {
		return
	}

	a.poling()
}

func (a *Api) Stop() {
	a.cancel()
}

func (a *Api) ProcessUpdate(_ context.Context, u model.Update) {
	a.ProcessContext(NewContext(a.ctx, a.client, u))
}

func (a *Api) ProcessContext(c Context) {
	u := c.Update()

	if a.handle(callbackPrefix+a.getCallback(u), c) {
		return
	}

	if a.handle(a.getCommand(u), c) {
		return
	}

	if a.handle(string(u.UpdateType), c) {
		return
	}
}

func (a *Api) WebhookHandler() http.HandlerFunc {
	return a.client.GetHandler(a.ProcessUpdate, a.conf.secret)
}

func (a *Api) Client() *maxbot.Api {
	return a.client
}

func (a *Api) webhookSubscribe() error {
	if a.conf.webhookURL == "" || a.conf.secret == "" {
		return nil
	}

	result, rErr := a.client.Subscriptions.GetSubscriptions(a.ctx)
	if rErr != nil {
		return fmt.Errorf("cannot get subscriptions: %w", rErr)
	}

	for _, s := range result.Subscriptions {
		_, err := a.client.Subscriptions.Unsubscribe(a.ctx, s.URL)
		if err != nil {
			continue
		}
	}

	_, err := a.client.Subscriptions.Subscribe(a.ctx, a.conf.webhookURL, a.conf.secret, a.conf.types, "1")

	return err
}

func (a *Api) poling() {
	var updates []model.Update
	var marker int64
	var err error
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
		}

		updates, marker, err = a.client.Subscriptions.GetUpdates(a.ctx, marker)
		err = checkError(err)
		if err != nil {
			return
		}

		for _, update := range updates {
			a.ProcessUpdate(a.ctx, update)
		}
	}
}

func (a *Api) handle(end string, c Context) bool {
	if handler, ok := a.handlers[end]; ok {
		a.runHandler(handler, c)
		return true
	}
	return false
}

func (a *Api) runHandler(h HandlerFunc, c Context) {
	f := func() {
		if err := h(c); err != nil {
			a.OnError(err, c)
		}
	}

	go f()
}

func (a *Api) OnError(err error, c Context) {
	a.onError(err, c)
}

func (a *Api) getCommand(u model.Update) string {
	command := u.GetCommand()

	// В личной переписке не может быть других ботов
	if u.GetMessage().Recipient.ChatType == model.ChatTypeDialog {
		return command.Command
	}

	// в групповых чатах, без указания бота
	// тут мы не можем проверить наличие других ботов,
	// поэтому делаем предположение, что бот единственный и, если не тегнули,
	// то отвечаем на все команды
	if command.BotName == "" {
		return command.Command
	}

	if command.BotName == a.Info.Username {
		return command.Command
	}

	return ""
}

func (a *Api) getCallback(u model.Update) string {
	callback := u.GetCallbackPayload()

	return callback.Payload
}

func defaultOnError(err error, c Context) {
	if c != nil {
		log.Println(c.Update().ChatID, err)
		return
	}

	log.Println(err)
}

func checkError(err error) error {
	if err == nil {
		return nil
	}

	// Обрабатывать timeout как пустую страницу (ожидается при длительном опросе)
	if _, ok := errors.AsType[*maxbot.TimeoutError](err); ok {
		return nil
	}

	return err
}
