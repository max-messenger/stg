package bot

import (
	"context"
	"time"

	"go.uber.org/zap"

	"stg/internal/app/domain"
	botservice "stg/internal/app/services/bot"
	"stg/pkg/locker"
)

type Harness struct {
	Bot    *botservice.MaxBot
	Client *ClientSpy
}

type NoopStore struct{}

func (NoopStore) GetState(_ context.Context, _ int64) (string, error) { return "", nil }
func (NoopStore) SetState(_ context.Context, _ int64, _ string) error { return nil }
func (NoopStore) ClearState(_ context.Context, _ int64) error         { return nil }

type NoopRepository struct{}

func (NoopRepository) Add(_ context.Context, _ domain.Todo) (int64, error) { return 0, nil }
func (NoopRepository) List(_ context.Context, _ int64) ([]domain.Todo, error) {
	return nil, nil
}
func (NoopRepository) Delete(_ context.Context, _ domain.Todo) error { return nil }

type NoopLocker struct{}

func (NoopLocker) Lock(_ context.Context, _ locker.Key, _ string, _ time.Duration) (func(context.Context) error, error) {
	return func(context.Context) error { return nil }, nil
}

func NewHarness(
	l *zap.Logger,
	store botservice.Store,
	repo botservice.Repository,
) Harness {
	if l == nil {
		l = zap.NewNop()
	}
	if store == nil {
		store = NoopStore{}
	}
	if repo == nil {
		repo = NoopRepository{}
	}
	clientSpy := NewClientSpy()

	return Harness{
		Bot:    botservice.New(l, clientSpy, store, repo),
		Client: clientSpy,
	}
}
