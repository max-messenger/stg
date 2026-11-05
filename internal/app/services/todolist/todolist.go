package todolist

import (
	"context"
	"fmt"

	"stg/internal/app/domain"
)

const messageLimit = 100

//go:generate go tool mockgen -source=todolist.go -destination=todolist_mocks_test.go  -package=todolist_test
type Repository interface {
	Add(ctx context.Context, d domain.Todo) (int64, error)
	Edit(ctx context.Context, d domain.Todo) (int64, error)
	List(ctx context.Context, userID int64) ([]domain.Todo, error)
	Delete(ctx context.Context, d domain.Todo) error
	Toggle(ctx context.Context, d domain.Todo) error
}

type Auth interface {
	ValidateUserID(ctx context.Context, userID int64) error
}

type Service struct {
	repo Repository
	auth Auth
}

func NewService(r Repository, a Auth) *Service {
	return &Service{
		repo: r,
		auth: a,
	}
}

func (s *Service) Create(ctx context.Context, d domain.Todo) (int64, error) {
	err := s.auth.ValidateUserID(ctx, d.UserID)
	if err != nil {
		return 0, fmt.Errorf("create error: %w", err)
	}

	if len([]rune(d.Message)) > messageLimit {
		return 0, fmt.Errorf("text length exceeds maximum allowed. Maximum is 100 characters")
	}
	id, err := s.repo.Add(ctx, d)

	return id, err
}

func (s *Service) Edit(ctx context.Context, d domain.Todo) (int64, error) {
	err := s.auth.ValidateUserID(ctx, d.UserID)
	if err != nil {
		return 0, fmt.Errorf("edit error: %w", err)
	}

	if len([]rune(d.Message)) > messageLimit {
		return 0, fmt.Errorf("text length exceeds maximum allowed. Maximum is 100 characters")
	}
	id, err := s.repo.Edit(ctx, d)

	return id, err
}

func (s *Service) Delete(ctx context.Context, d domain.Todo) error {
	err := s.auth.ValidateUserID(ctx, d.UserID)
	if err != nil {
		return fmt.Errorf("create error: %w", err)
	}

	err = s.repo.Delete(ctx, d)

	return err
}

func (s *Service) Toggle(ctx context.Context, d domain.Todo) error {
	err := s.auth.ValidateUserID(ctx, d.UserID)
	if err != nil {
		return fmt.Errorf("toggle error: %w", err)
	}

	err = s.repo.Toggle(ctx, d)

	return err
}

func (s *Service) List(ctx context.Context, userID int64) ([]domain.Todo, error) {
	err := s.auth.ValidateUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list error: %w", err)
	}

	result, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	return result, nil
}
