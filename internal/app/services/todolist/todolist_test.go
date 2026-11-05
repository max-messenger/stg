package todolist_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"stg/internal/app/domain"
	"stg/internal/app/services/todolist"
)

func TestUnitTodoService(t *testing.T) {
	suite.Run(t, new(testService))
}

type testService struct {
	suite.Suite

	service    *todolist.Service
	repository *MockRepository
	auth       *MockAuth
}

func (t *testService) SetupTest() {
	ctrl := gomock.NewController(t.T())

	t.repository = NewMockRepository(ctrl)
	t.auth = NewMockAuth(ctrl)
	t.service = todolist.NewService(t.repository, t.auth)
}

func (t *testService) TestAddSuccess() {
	ctx := context.Background()
	d := domain.Todo{
		UserID:  1,
		Message: "test",
		Done:    false,
		Created: time.Now(),
	}
	expectedID := int64(1) // Simulate a valid database ID

	t.repository.EXPECT().
		Add(ctx, d).
		Return(expectedID, nil)

	t.auth.EXPECT().ValidateUserID(ctx, d.UserID)

	id, err := t.service.Create(ctx, d)
	t.NoError(err)
	assert.Equal(t.T(), expectedID, id)
}

func (t *testService) TestListSuccess() {
	ctx := context.Background()
	d := []domain.Todo{
		{
			Message: "test",
			Done:    false,
			Created: time.Now(),
		},
	}

	t.repository.EXPECT().List(ctx, gomock.Any()).Return(d, nil)
	t.auth.EXPECT().ValidateUserID(ctx, int64(1))

	res, err := t.service.List(ctx, 1)
	t.NoError(err)

	t.Equal(res, d)
}

func (t *testService) TestDeleteSuccess() {
	ctx := context.Background()
	d := domain.Todo{
		ID:      1,
		UserID:  1,
		Message: "test",
		Done:    false,
		Created: time.Now(),
	}

	t.repository.EXPECT().Delete(ctx, d).Return(nil)
	t.auth.EXPECT().ValidateUserID(ctx, d.UserID)

	err := t.service.Delete(ctx, d)
	t.NoError(err)
}

func (t *testService) TestToggleSuccess() {
	ctx := context.Background()
	d := domain.Todo{
		ID:      1,
		UserID:  1,
		Message: "test",
		Done:    true,
		Created: time.Now(),
	}

	t.repository.EXPECT().Toggle(ctx, d).Return(nil)
	t.auth.EXPECT().ValidateUserID(ctx, d.UserID)

	err := t.service.Toggle(ctx, d)
	t.NoError(err)
}

func (t *testService) TestEditSuccess() {
	ctx := context.Background()
	d := domain.Todo{
		ID:      1,
		UserID:  1,
		Message: "updated message",
		Done:    false,
		Created: time.Now(),
	}
	expectedID := int64(1)

	t.repository.EXPECT().
		Edit(ctx, d).
		Return(expectedID, nil)

	t.auth.EXPECT().ValidateUserID(ctx, d.UserID)

	id, err := t.service.Edit(ctx, d)
	t.NoError(err)
	assert.Equal(t.T(), expectedID, id)
}
