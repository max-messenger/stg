package domain

import "github.com/max-messenger/max-bot-api-client-go/v2/model"

type Message struct {
	UserID      int64
	ChatID      int64
	Text        string
	Keyboard    *model.Keyboard
	Attachments []*model.Attachment
}
