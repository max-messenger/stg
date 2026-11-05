package subscriber

import (
	"context"
	"sync"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type (
	Messages interface {
		Updates() <-chan model.Update
	}

	Bot interface {
		ProcessUpdate(context.Context, model.Update)
	}
)

type Subscriber struct {
	logger      *zap.Logger
	botMessages Messages
	bot         Bot
	quit        chan struct{}
	wg          sync.WaitGroup
}

func New(
	logger *zap.Logger,
	botMessages Messages,
	bot Bot,
) *Subscriber {
	return &Subscriber{
		logger:      logger,
		botMessages: botMessages,
		bot:         bot,

		quit: make(chan struct{}),
	}
}

func (s *Subscriber) Start(_ context.Context) error {
	s.logger.Info("update subscriber started")
	s.wg.Go(func() {
		s.messagesConsumer()
	})

	return nil
}

func (s *Subscriber) Stop(_ context.Context) error {
	s.logger.Info("update subscriber stopping")
	close(s.quit)
	s.wg.Wait()
	s.logger.Info("update subscriber stopped")

	return nil
}
func (s *Subscriber) messagesConsumer() {
	ch := s.botMessages.Updates()
	for {
		select {
		case <-s.quit:
			return
		case upd, ok := <-ch:
			if !ok {
				s.logger.Info("update channel closed")

				return // channel closed
			}
			s.logger.Debug("dispatching update",
				zap.String("update_type", string(upd.UpdateType)),
				zap.Int64("user_id", upd.UserID),
				zap.Int64("chat_id", upd.ChatID),
			)
			// крутим всё горутинах, чтобы не блокироваться
			s.wg.Go(func() {
				ctx, span := otel.Tracer("subscriber").Start(context.Background(), "bot-message")
				defer span.End()

				defer func(t time.Time) {
					metricRequestsTotal.WithLabelValues("bot-message").Inc()
					metricRequestsDuration.WithLabelValues("bot-message").Observe(time.Since(t).Seconds())
				}(time.Now())

				s.bot.ProcessUpdate(ctx, upd)
			})
		}
	}

}
