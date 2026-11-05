package bot_test

import (
	"context"
	"testing"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"stg/internal/app/domain"
	botservice "stg/internal/app/services/bot"
	"stg/internal/app/services/bot/mocks"
	testbot "stg/testkit/bot"
)

const infoMessage = `/info - info
/create - new record
/list - all record
/delete - delete record
/myid - get max id`

type testDeps struct {
	ctx     context.Context
	client  *testbot.ClientSpy
	store   *mocks.MockStore
	repo    *mocks.MockRepository
	service *botservice.MaxBot
}

func prepare(t *testing.T) *testDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	clientSpy := testbot.NewClientSpy()
	storeMock := mocks.NewMockStore(ctrl)
	repoMock := mocks.NewMockRepository(ctrl)

	bot := botservice.New(
		zap.NewNop(),
		clientSpy,
		storeMock,
		repoMock,
	)

	return &testDeps{
		ctx:     context.Background(),
		client:  clientSpy,
		store:   storeMock,
		repo:    repoMock,
		service: bot,
	}
}

func TestUnitProcessUpdate_MessageCreated_HelpCommand(t *testing.T) {
	d := prepare(t)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("/help").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	testbot.RequireSentCount(t, d.client, 1)
	testbot.RequireLastSentText(t, d.client, infoMessage)
}

func TestUnitProcessUpdate_BotStarted_SendsInfo(t *testing.T) {
	d := prepare(t)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).BotStarted()
	d.service.ProcessUpdate(d.ctx, upd)

	testbot.RequireSentCount(t, d.client, 1)
	testbot.RequireLastSentText(t, d.client, infoMessage)
}

func TestUnitProcessUpdate_MessageCreated_CreateCommand(t *testing.T) {
	d := prepare(t)

	d.store.EXPECT().SetState(gomock.Any(), int64(42), "state_create").Return(nil)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("/create").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	testbot.RequireSentCount(t, d.client, 1)
	testbot.RequireLastSentText(t, d.client, "Введите текст новой записи")
}

func TestUnitProcessUpdate_MessageCreated_ListCommand_WithRecords(t *testing.T) {
	d := prepare(t)

	createdAt := time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC)
	d.repo.EXPECT().List(gomock.Any(), int64(42)).Return([]domain.Todo{
		{Created: createdAt, Message: "first"},
		{Created: createdAt.Add(time.Minute), Message: "second"},
	}, nil)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("/list").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	msg, ok := d.client.LastSentMessage()
	require.True(t, ok)
	require.Contains(t, msg.Text, "first")
	require.Contains(t, msg.Text, "second")
}

func TestUnitProcessUpdate_MessageCreated_DeleteCommand_WithRecords(t *testing.T) {
	d := prepare(t)

	d.repo.EXPECT().List(gomock.Any(), int64(42)).Return([]domain.Todo{
		{ID: 9, Created: time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC), Message: "todo-to-delete"},
	}, nil)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("/delete").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	msg, ok := d.client.LastSentMessage()
	require.True(t, ok)
	require.Equal(t, "Выберите запись для удаления:", msg.Text)
	require.NotNil(t, msg.Keyboard)
}

func TestUnitProcessUpdate_MessageCreated_StateCreate_AddRecord(t *testing.T) {
	d := prepare(t)

	d.store.EXPECT().GetState(gomock.Any(), int64(42)).Return("state_create", nil)
	d.repo.EXPECT().
		Add(gomock.Any(), gomock.AssignableToTypeOf(domain.Todo{})).
		DoAndReturn(func(_ context.Context, todo domain.Todo) (int64, error) {
			require.Equal(t, int64(42), todo.UserID)
			require.Equal(t, "new todo text", todo.Message)
			return int64(1), nil
		})

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("new todo text").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	testbot.RequireLastSentText(t, d.client, "Запись создана")
}

func TestUnitProcessUpdate_MessageCreated_MyIDCommand(t *testing.T) {
	d := prepare(t)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("/myid").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	msg, ok := d.client.LastSentMessage()
	require.True(t, ok)
	require.Equal(t, "name: test-user\nuser_id: 42\nchat_id: 77", msg.Text)
}

func TestUnitProcessUpdate_MessageCallback_Delete_WithEdit(t *testing.T) {
	d := prepare(t)
	d.repo.EXPECT().Delete(gomock.Any(), domain.Todo{ID: 10, UserID: 42}).Return(nil)
	d.repo.EXPECT().List(gomock.Any(), int64(42)).Return([]domain.Todo{
		{ID: 11, Created: time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC), Message: "still exists"},
	}, nil)

	upd := testbot.NewUpdateBuilder().
		WithUserID(42).
		WithChatID(77).
		WithCallbackPayload("/delete:10").
		WithCallbackMessageID("mid-42").
		MessageCallback()
	d.service.ProcessUpdate(d.ctx, upd)

	require.Len(t, d.client.EditCalls(), 1)
	require.Len(t, d.client.DeleteCalls(), 0)
}

func TestUnitProcessUpdate_MessageCallback_Delete_WithDeleteMessage(t *testing.T) {
	d := prepare(t)
	d.repo.EXPECT().Delete(gomock.Any(), domain.Todo{ID: 10, UserID: 42}).Return(nil)
	d.repo.EXPECT().List(gomock.Any(), int64(42)).Return([]domain.Todo{}, nil)

	upd := testbot.NewUpdateBuilder().
		WithUserID(42).
		WithChatID(77).
		WithCallbackPayload("/delete:10").
		WithCallbackMessageID("mid-42").
		MessageCallback()
	d.service.ProcessUpdate(d.ctx, upd)

	require.Len(t, d.client.DeleteCalls(), 1)
	require.Equal(t, "mid-42", d.client.DeleteCalls()[0].MID)
	require.Len(t, d.client.EditCalls(), 0)
}

func TestUnitProcessUpdate_UnknownUpdate_NoOutgoingMessages(t *testing.T) {
	d := prepare(t)

	d.service.ProcessUpdate(d.ctx, model.Update{UserID: 42, ChatID: 77, UpdateType: model.UpdateBotAdded})

	require.Len(t, d.client.SentCalls(), 0)
	require.Len(t, d.client.EditCalls(), 0)
	require.Len(t, d.client.DeleteCalls(), 0)
}

func TestUnitProcessUpdate_DefaultState_SendsCommandText(t *testing.T) {
	d := prepare(t)

	d.store.EXPECT().GetState(gomock.Any(), int64(42)).Return("unknown_state", nil)

	upd := testbot.NewUpdateBuilder().WithUserID(42).WithChatID(77).WithText("hello").MessageCreated()
	d.service.ProcessUpdate(d.ctx, upd)

	msg, ok := d.client.LastSentMessage()
	require.True(t, ok)
	require.Contains(t, msg.Text, "команда")
}

func TestUnitProcessUpdate_MessageCallback_DefaultHandler(t *testing.T) {
	d := prepare(t)

	upd := testbot.NewUpdateBuilder().
		WithUserID(42).
		WithChatID(77).
		WithCallbackPayload("some-unknown-payload").
		MessageCallback()
	d.service.ProcessUpdate(d.ctx, upd)

	require.Len(t, d.client.SentCalls(), 0)
	require.Len(t, d.client.EditCalls(), 0)
	require.Len(t, d.client.DeleteCalls(), 0)
}
