package bot

import (
	"context"
	"fmt"
	"sync"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"

	"stg/internal/app/domain"
)

type SentCall struct {
	Ctx context.Context
	Msg domain.Message
}

type EditCall struct {
	Ctx context.Context
	MID string
	Msg domain.Message
}

type DeleteCall struct {
	Ctx context.Context
	MID string
}

// ClientSpy stores all outbound calls made by bot service.
type ClientSpy struct {
	mu sync.RWMutex

	sendCalls   []SentCall
	editCalls   []EditCall
	deleteCalls []DeleteCall
}

func NewClientSpy() *ClientSpy {
	return &ClientSpy{}
}

func (c *ClientSpy) Updates() <-chan model.Update {
	return nil
}

func (c *ClientSpy) SendMessage(ctx context.Context, msg domain.Message) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sendCalls = append(c.sendCalls, SentCall{
		Ctx: ctx,
		Msg: msg,
	})

	return fmt.Sprintf("spy-mid-%d", len(c.sendCalls)), nil
}

func (c *ClientSpy) EditMessage(ctx context.Context, mid string, msg domain.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.editCalls = append(c.editCalls, EditCall{
		Ctx: ctx,
		MID: mid,
		Msg: msg,
	})

	return nil
}

func (c *ClientSpy) DeleteMessage(ctx context.Context, mid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteCalls = append(c.deleteCalls, DeleteCall{
		Ctx: ctx,
		MID: mid,
	})

	return nil
}

func (c *ClientSpy) SentCalls() []SentCall {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]SentCall, len(c.sendCalls))
	copy(out, c.sendCalls)

	return out
}

func (c *ClientSpy) EditCalls() []EditCall {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]EditCall, len(c.editCalls))
	copy(out, c.editCalls)

	return out
}

func (c *ClientSpy) DeleteCalls() []DeleteCall {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]DeleteCall, len(c.deleteCalls))
	copy(out, c.deleteCalls)

	return out
}

func (c *ClientSpy) LastSentMessage() (domain.Message, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.sendCalls) == 0 {
		return domain.Message{}, false
	}

	return c.sendCalls[len(c.sendCalls)-1].Msg, true
}
