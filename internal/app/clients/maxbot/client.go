//nolint:revive
package maxbot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	maxbotcli "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"go.uber.org/zap"

	"stg/internal/app/domain"
	"stg/pkg/telemetry"
)

const (
	subscribeTypeWebhook = "webhook"
	subscribeTypePoll    = "poll"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	logger *zap.Logger
	config Config

	client *maxbotcli.Api

	ctx    context.Context
	cancel context.CancelFunc

	updates chan model.Update
}

func New(logger *zap.Logger, config Config, cli HTTPClient) (*Client, error) {
	opts := []maxbotcli.Opt{
		maxbotcli.WithHTTPClient(cli),
	}
	if config.isTest() {
		opts = append(opts, maxbotcli.WithBaseURL(config.TestApiUrl))
	}

	maxCli, err := maxbotcli.NewApi(config.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("create max client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		client: maxCli,
		logger: logger,
		config: config,

		// for polling
		ctx:    ctx,
		cancel: cancel,

		updates: make(chan model.Update, 256),
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	info, err := c.client.Bots.GetMyInfo(ctx)
	if err != nil {
		return fmt.Errorf("get bot info: %w", err)
	}

	c.logger.Info("bot", zap.Any("info", info))

	err = c.unsubscribe(ctx)
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}

	c.logger.Info("subscribe", zap.String("type", c.config.SubscriptionType))

	err = c.subscribe(ctx)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	return nil
}

func (c *Client) Stop(_ context.Context) error {
	c.cancel()
	close(c.updates)

	return nil
}

func (c *Client) SendMessage(ctx context.Context, msg domain.Message) (string, error) {
	var err error
	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("send-message", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("send-message").Observe(time.Since(t).Seconds())
	}(time.Now())

	res, err := c.client.Messages.Send(ctx, makeMessage(msg))
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	return res.Message.Body.Mid, nil
}

func (c *Client) GetMessage(ctx context.Context, mid string) (model.Message, error) {
	var err error
	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("get-message", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("get-message").Observe(time.Since(t).Seconds())
	}(time.Now())

	message, err := c.client.Messages.GetMessageByID(ctx, mid)
	if err != nil {
		return model.Message{}, fmt.Errorf("get message by id: %w", err)
	}

	return message, nil
}

func (c *Client) EditMessage(ctx context.Context, mid string, msg domain.Message) error {
	var err error
	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("edit-message", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("edit-message").Observe(time.Since(t).Seconds())
	}(time.Now())

	_, err = c.client.Messages.EditMessage(ctx, mid, makeMessage(msg).MessageBody())
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return nil
}

func (c *Client) DeleteMessage(ctx context.Context, mid string) error {
	var err error
	defer func(t time.Time) {
		metricRequestsTotal.WithLabelValues("delete-message", telemetry.ErrLabelValue(err)).Inc()
		metricRequestsDuration.WithLabelValues("delete-message").Observe(time.Since(t).Seconds())
	}(time.Now())

	_, err = c.client.Messages.DeleteMessage(ctx, mid)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return nil
}
