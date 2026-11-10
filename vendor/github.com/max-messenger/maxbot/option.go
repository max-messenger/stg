package maxbot

import (
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

type Option func(msg *maxbot.Message)

func WithKeyboard(keyboard *model.Keyboard) Option {
	return func(msg *maxbot.Message) {
		msg.AddKeyboard(keyboard)
	}
}

func WithFormat(format model.TextFormat) Option {
	return func(msg *maxbot.Message) {
		msg.SetFormat(format)
	}
}

func WithChat(id int64) Option {
	return func(msg *maxbot.Message) {
		msg.SetChat(id)
	}
}

func WithUser(id int64) Option {
	return func(msg *maxbot.Message) {
		msg.SetUser(id)
	}
}

func WithDisableLinkPreview(mode bool) Option {
	return func(msg *maxbot.Message) {
		msg.SetDisableLinkPreview(mode)
	}
}

func WithAddImageUr(url string) Option {
	return func(msg *maxbot.Message) {
		msg.AddImageUrl(url)
	}
}

func WithSticker(stickerCode string) Option {
	return func(msg *maxbot.Message) {
		msg.AddSticker(stickerCode)
	}
}

func WithContact(userID int64) Option {
	return func(msg *maxbot.Message) {
		msg.AddContact(userID)
	}
}

func WithLocation(lat, lon float64) Option {
	return func(msg *maxbot.Message) {
		msg.AddLocation(lat, lon)
	}
}

func WithShare(link string) Option {
	return func(msg *maxbot.Message) {
		msg.AddShare(link)
	}
}

func WithoutNotify() Option {
	return func(msg *maxbot.Message) {
		msg.WithoutNotify()
	}
}

func WithReply(text, id string) Option {
	return func(msg *maxbot.Message) {
		msg.SetReply(text, id)
	}
}

// WithAttachByToken создает опцию для добавления вложения к сообщению по токену файла.
//
// Используется паттерн функциональных опций (functional options pattern) для гибкого
// конструирования сообщений с вложениями различных типов (фото, видео, документы и т.д.).
//
// Параметры:
//   - fileToken: строковый токен файла (например, URL или идентификатор загруженного файла)
//   - at: тип вложения из модели AttachmentType (Photo, Video, Document и др.)
//
// Возвращает:
//
//	Option — функцию-модификатор, которая добавляет вложение к сообщению.
//
// Пример использования:
//
//		// Добавление фото к сообщению
//	 msg := maxbot.NewMessage()
//	 msg.WithAttachByToken("photo_token_123",  model.AttachImage)
//
//		// Или через цепочку опций при построении сообщения
//		message := BuildMessage(
//		    WithText("Привет!"),
//		    WithAttachByToken("file_token", model.AttachImage),
//		)
func WithAttachByToken(fileToken string, at model.AttachmentType) Option {
	return func(msg *maxbot.Message) {
		msg.AddAttachByToken(fileToken, at)
	}
}

// WithAttachments прокидывает все вложения сообщения в методах Edit, Answer.
// По умолчанию методы Edit, Answer затирают все кнопки и пр. вложения.
func WithAttachments(attachments []model.Attachment) Option {
	return func(msg *maxbot.Message) {
		msg.AddAttachments(attachments)
	}
}

func WithMessageID(messageID string) Option {
	return func(msg *maxbot.Message) {
		msg.MessageID = messageID
	}
}
