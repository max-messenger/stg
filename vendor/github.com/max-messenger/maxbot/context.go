package maxbot

import (
	"context"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

type HandlerFunc func(Context) error

type Context interface {
	Update() model.Update
	Context() context.Context
	WithValue(ctx context.Context, key, value any)
	API() *maxbot.Api

	Send(text string, opts ...Option) error
	SendMessage(*maxbot.Message) error
	Answer(text string, opts ...Option) error
	Reply(text string, opts ...Option) error
	Edit(text string, opts ...Option) error
	Delete(opts ...Option) error
}

type nativeContext struct {
	ctx context.Context
	b   *maxbot.Api
	u   model.Update
	box map[string]any
}

func NewContext(ctx context.Context, b *maxbot.Api, u model.Update) Context {
	return &nativeContext{
		ctx: ctx,
		b:   b,
		u:   u,
	}
}

func (c *nativeContext) Update() model.Update {
	return c.u
}

func (c *nativeContext) Context() context.Context {
	return c.ctx
}

func (c *nativeContext) WithValue(ctx context.Context, key, value any) {
	c.ctx = context.WithValue(ctx, key, value)
}

func (c *nativeContext) API() *maxbot.Api {
	return c.b
}

func (c *nativeContext) Send(text string, opts ...Option) error {
	msg := maxbot.NewMessage().
		SetText(text).
		SetUser(c.u.UserID).
		SetChat(c.u.ChatID).SetFormat(model.FormatMarkdown)

	for _, opt := range opts {
		opt(msg)
	}

	_, err := c.b.Messages.Send(c.ctx, msg)

	return err
}

func (c *nativeContext) SendMessage(msg *maxbot.Message) error {
	_, err := c.b.Messages.Send(c.ctx, msg)

	return err
}

func (c *nativeContext) Answer(text string, opts ...Option) error {
	msg := maxbot.NewMessage().
		SetText(text).
		SetUser(c.u.UserID).
		SetChat(c.u.ChatID)

	for _, opt := range opts {
		opt(msg)
	}

	mb := msg.MessageBody()
	_, err := c.b.Messages.AnswerOnCallback(c.ctx, c.u.GetCallback().CallbackID, model.CallbackAnswer{Message: &mb})

	return err
}

func (c *nativeContext) Reply(text string, opts ...Option) error {
	msg := maxbot.NewMessage().
		SetUser(c.u.UserID).
		SetChat(c.u.ChatID).
		SetReply(text, c.u.GetMessage().Body.Mid)

	for _, opt := range opts {
		opt(msg)
	}

	_, err := c.b.Messages.Send(c.ctx, msg)

	return err
}

func (c *nativeContext) Edit(text string, opts ...Option) error {
	msg := maxbot.NewMessage().
		SetText(text).
		SetUser(c.u.UserID).
		SetChat(c.u.ChatID)

	msg.MessageID = c.u.MessageID

	for _, opt := range opts {
		opt(msg)
	}

	_, err := c.b.Messages.EditMessage(c.ctx, msg.MessageID, msg.MessageBody())

	return err
}

func (c *nativeContext) Delete(opts ...Option) error {
	msg := maxbot.NewMessage().
		SetUser(c.u.UserID).
		SetChat(c.u.ChatID)

	msg.MessageID = c.u.MessageID

	for _, opt := range opts {
		opt(msg)
	}

	_, err := c.b.Messages.DeleteMessage(c.ctx, msg.MessageID)

	return err
}
