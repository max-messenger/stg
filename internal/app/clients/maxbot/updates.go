package maxbot

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	maxbotcli "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"go.uber.org/zap"

	"stg/internal/app/domain"
)

func (c *Client) UpdateHandler() (string, http.HandlerFunc) {
	updateHandler := func(_ context.Context, upd model.Update) {
		c.updates <- upd
	}

	maxHandler := c.client.GetHandler(updateHandler, c.config.Secret)

	return c.config.Path, maxHandler
}

func (c *Client) Updates() <-chan model.Update {
	if c.config.SubscriptionType == subscribeTypePoll {
		return c.getUpdates()
	}

	return c.updates
}

func (c *Client) getUpdates() <-chan model.Update {
	go func() {
		var updates []model.Update
		var marker int64
		var err error
		for {
			select {
			case <-c.ctx.Done():
			default:
				updates, marker, err = c.client.Subscriptions.GetUpdates(c.ctx, marker)
				err = c.checkError(err)
				if err != nil {
					c.logger.Error("get updates", zap.Error(err))

					return
				}

				for _, update := range updates {
					c.updates <- update
				}
			}
		}
	}()

	return c.updates
}

func (c *Client) checkError(err error) error {
	if err == nil {
		return nil
	}

	// Обрабатывать timeout как пустую страницу (ожидается при длительном опросе)
	if _, ok := errors.AsType[*maxbotcli.TimeoutError](err); ok { //nolint:errcheck
		return nil
	}

	return err
}

func (c *Client) unsubscribe(ctx context.Context) error {
	result, rErr := c.client.Subscriptions.GetSubscriptions(ctx)
	if rErr != nil {
		return fmt.Errorf("cannot get subscriptions: %w", rErr)
	}

	for _, s := range result.Subscriptions {
		c.logger.Info("unsubscribe", zap.Any("subscription", s))
		res, err := c.client.Subscriptions.Unsubscribe(ctx, s.URL)
		if err != nil {
			c.logger.Error("cannot unsubscribe", zap.Error(err))

			continue
		}
		c.logger.Info("unsubscribe", zap.Any("response", res))
	}

	return nil
}

func (c *Client) subscribe(ctx context.Context) error {
	// subscribe only for webhook
	if c.config.SubscriptionType != subscribeTypeWebhook {
		return nil
	}

	webhookURL, err := url.JoinPath(c.config.Url, c.config.Path)
	if err != nil {
		return fmt.Errorf("join webhook url: %w", err)
	}

	subscriptionResp, err := c.client.Subscriptions.Subscribe(ctx, webhookURL, c.config.Secret, c.config.Types, "1")
	if err != nil {
		return fmt.Errorf("cannot subscribe: %w", err)
	}

	c.logger.Info("subscribe", zap.Any("response", subscriptionResp))

	return nil
}

func makeMessage(m domain.Message) *maxbotcli.Message {
	result := maxbotcli.NewMessage().
		SetUser(m.UserID).SetChat(m.ChatID).SetText(m.Text)

	if m.Text != "" {
		result.SetFormat("markdown")
	}

	result.AddKeyboard(m.Keyboard)

	return result
}
