package bot

import (
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

const defaultTimestamp = 1700000000000
const defaultTestUserName = "test-user"

// UpdateBuilder helps compose MAX updates for unit-tests.
type UpdateBuilder struct {
	userID    int64
	chatID    int64
	text      string
	updateTS  int
	callback  string
	callbackM string
}

func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{
		userID:   1001,
		chatID:   2001,
		text:     "/help",
		updateTS: defaultTimestamp,
	}
}

func (b *UpdateBuilder) WithUserID(v int64) *UpdateBuilder {
	b.userID = v

	return b
}

func (b *UpdateBuilder) WithChatID(v int64) *UpdateBuilder {
	b.chatID = v

	return b
}

func (b *UpdateBuilder) WithText(v string) *UpdateBuilder {
	b.text = v

	return b
}

func (b *UpdateBuilder) WithTimestamp(v time.Time) *UpdateBuilder {
	b.updateTS = int(v.UnixMilli())

	return b
}

func (b *UpdateBuilder) WithCallbackPayload(v string) *UpdateBuilder {
	b.callback = v

	return b
}

func (b *UpdateBuilder) WithCallbackMessageID(v string) *UpdateBuilder {
	b.callbackM = v

	return b
}

func (b *UpdateBuilder) MessageCreated() model.Update {
	return model.Update{
		UpdateType: model.UpdateMessageCreated,
		Timestamp:  int64(b.updateTS),
		UserID:     b.userID,
		ChatID:     b.chatID,
		Message: &model.MessageUpdate{
			Sender: model.Sender{Name: defaultTestUserName},
			Body:   model.MessageBody{Mid: "mid-1", Text: b.text},
		},
	}
}

func (b *UpdateBuilder) BotStarted() model.Update {
	return model.Update{
		UpdateType: model.UpdateBotStarted,
		Timestamp:  int64(b.updateTS),
		UserID:     b.userID,
		ChatID:     b.chatID,
		User:       &model.User{UserID: b.userID, Name: defaultTestUserName},
	}
}

func (b *UpdateBuilder) MessageCallback() model.Update {
	callbackMID := b.callbackM
	if callbackMID == "" {
		callbackMID = "mid-callback-1"
	}

	return model.Update{
		UpdateType: model.UpdateMessageCallback,
		Timestamp:  int64(b.updateTS),
		UserID:     b.userID,
		ChatID:     b.chatID,
		Callback: &model.Callback{
			Timestamp:  int64(b.updateTS),
			CallbackID: "cb-1",
			Payload:    b.callback,
			User:       model.User{UserID: b.userID, Name: defaultTestUserName},
		},
		Message: &model.MessageUpdate{Body: model.MessageBody{Mid: callbackMID}},
	}
}
