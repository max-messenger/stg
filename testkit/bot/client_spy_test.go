package bot_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stg/internal/app/domain"
	testbot "stg/testkit/bot"
)

func TestClientSpy_SendAndReadLast(t *testing.T) {
	spy := testbot.NewClientSpy()
	_, err := spy.SendMessage(context.Background(), domain.Message{Text: "first"})
	require.NoError(t, err)

	_, err = spy.SendMessage(context.Background(), domain.Message{Text: "second"})
	require.NoError(t, err)

	testbot.RequireSentCount(t, spy, 2)
	testbot.RequireLastSentText(t, spy, "second")
}

func TestUpdateBuilder_MessageCreated(t *testing.T) {
	upd := testbot.NewUpdateBuilder().
		WithUserID(11).
		WithChatID(22).
		WithText("/help").
		MessageCreated()

	require.Equal(t, int64(11), upd.UserID)
	require.Equal(t, int64(22), upd.ChatID)
	require.Equal(t, "/help", upd.Message.Body.Text)
}
