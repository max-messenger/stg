package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"go.uber.org/zap"

	"stg/internal/app/domain"
)

type msgHandler func(context.Context, model.Update) (domain.Message, error)

const (
	commandHelp   = "/help"
	commandInfo   = "/info"
	commandCreate = "/create"
	commandList   = "/list"
	commandDelete = "/delete"
	commandMyID   = "/myid"

	stateCreate = "state_create"
	infoMessage = `/info - info
/create - new record
/list - all record
/delete - delete record
/myid - get max id`
)

//go:generate go tool mockgen -source=service.go -destination=mocks/service_mocks.go -package=mocks

type Repository interface {
	Add(ctx context.Context, d domain.Todo) (int64, error)
	List(ctx context.Context, userID int64) ([]domain.Todo, error)
	Delete(ctx context.Context, d domain.Todo) error
}

type Store interface {
	GetState(ctx context.Context, userID int64) (string, error)
	SetState(ctx context.Context, userID int64, value string) error
	ClearState(ctx context.Context, userID int64) error
}

type Client interface {
	Updates() <-chan model.Update
	SendMessage(ctx context.Context, msg domain.Message) (string, error)
	EditMessage(ctx context.Context, mid string, msg domain.Message) error
	DeleteMessage(ctx context.Context, mid string) error
}

type MaxBot struct {
	log *zap.Logger

	client Client
	store  Store
	repo   Repository
}

func New(l *zap.Logger, c Client, s Store, r Repository) *MaxBot {
	return &MaxBot{
		log:    l,
		client: c,
		store:  s,
		repo:   r,
	}
}

func (m *MaxBot) ProcessUpdate(ctx context.Context, upd model.Update) {
	m.log.Debug(
		"process update",
		zap.Any("user", upd.UserID),
		zap.Any("update_type", upd.UpdateType),
	)

	// nolint: exhaustive
	switch upd.UpdateType {
	case model.UpdateBotStarted:
		m.botStartedUpdate(ctx, upd)
	case model.UpdateMessageCreated:
		m.messageCreatedUpdate(ctx, upd)
	case model.UpdateMessageCallback:
		m.messageCallbackUpdate(ctx, upd)
	default:
		return
	}
}

func (m *MaxBot) messageCallbackUpdate(ctx context.Context, upd model.Update) {
	m.log.Info("Message callback update")

	if upd.Callback == nil || upd.Callback.Payload == "" {
		return
	}

	pd := domain.CallBack{}
	err := pd.Value(upd.Callback.Payload)
	if err != nil {
		m.log.Error("Failed to unmarshal callback payload", zap.Error(err))

		return
	}

	h := m.callbackDefaultHandler
	if pd.Key == commandDelete {
		h = m.callbackDeleteHandler
	}

	err = h(ctx, upd)
	if err != nil {
		m.log.Error("Failed to handle callback", zap.Error(err))
	}
}

func (m *MaxBot) messageCreatedUpdate(ctx context.Context, upd model.Update) { //nolint:cyclop
	m.log.Info("Message created update")

	stage, err := m.getState(ctx, upd)
	if err != nil {
		m.log.Error("", zap.Error(err))

		return
	}

	var h msgHandler
	switch stage {
	case commandHelp, commandInfo:
		h = m.helpStateHandler
	case commandCreate:
		h = m.commandCreateHandler
	case commandList:
		h = m.commandListHandler
	case stateCreate:
		h = m.stateCreateHandler
	case commandDelete:
		h = m.commandDeleteHandler
	case commandMyID:
		h = m.commandMyID
	default:
		h = m.defaultStateHandler
	}

	msg, err := h(ctx, upd)
	if err != nil {
		m.log.Warn("messageCreatedUpdate", zap.Error(err))
	}

	mid, err := m.client.SendMessage(ctx, msg)
	if err != nil {
		m.log.Warn("messageCreatedUpdate error", zap.Error(err))

		return
	}

	m.log.Debug("send message",
		zap.String("mid", mid),
		zap.Int64("chat_id", upd.ChatID),
		zap.Int64("user_id", upd.UserID),
	)
}

func (m *MaxBot) botStartedUpdate(ctx context.Context, upd model.Update) {
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text:   infoMessage,
	}

	_, err := m.client.SendMessage(ctx, msg)
	if err != nil {
		m.log.Warn("botStartedUpdate error", zap.Error(err))

		return
	}
}

func (m *MaxBot) getState(ctx context.Context, upd model.Update) (string, error) {
	if cmd := upd.GetCommand().Command; cmd != "" {
		return cmd, nil
	}

	state, err := m.store.GetState(ctx, upd.UserID)
	if err != nil {
		return state, fmt.Errorf("getState: %w", err)
	}

	return state, nil
}

func (m *MaxBot) defaultStateHandler(_ context.Context, upd model.Update) (domain.Message, error) {
	// /create
	out := "команда : " + upd.GetCommand().Command
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text:   out,
	}

	return msg, nil
}

func (m *MaxBot) helpStateHandler(_ context.Context, upd model.Update) (domain.Message, error) {
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text:   infoMessage,
	}

	return msg, nil
}

func (m *MaxBot) commandCreateHandler(ctx context.Context, upd model.Update) (domain.Message, error) {
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text:   "Введите текст новой записи",
	}

	err := m.store.SetState(ctx, upd.UserID, stateCreate)
	if err != nil {
		return msg, fmt.Errorf("setState: %w", err)
	}

	return msg, nil
}

func (m *MaxBot) commandListHandler(ctx context.Context, upd model.Update) (domain.Message, error) {
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text:   "У вас нет записей",
	}

	records, err := m.repo.List(ctx, msg.UserID)
	if err != nil {
		return msg, fmt.Errorf("list: %w", err)
	}

	fields := make([]string, 0, len(records))
	for _, record := range records {
		fields = append(fields, fmt.Sprintf("%s %s", record.Created.Format(time.DateTime), record.Message))
	}

	if len(fields) > 0 {
		msg.Text = strings.Join(fields, "\n")
	}

	return msg, nil
}

func (m *MaxBot) commandDeleteHandler(ctx context.Context, upd model.Update) (domain.Message, error) {
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text:   "У вас нет записей",
	}

	records, err := m.repo.List(ctx, msg.UserID)
	if err != nil {
		return msg, fmt.Errorf("list: %w", err)
	}

	if len(records) == 0 {
		return msg, nil
	}

	keyboard := model.NewKeyboard()
	for _, record := range records {
		txt := fmt.Sprintf("%s %s", record.Created.Format(time.DateTime), record.Message)
		pd := domain.CallBack{
			InternalID: record.ID,
			Key:        commandDelete,
		}
		keyboard.AddRow().AddCallBack(cutLimit(txt, 130), pd.String())
	}

	msg.Keyboard = keyboard
	msg.Text = "Выберите запись для удаления:"

	return msg, nil
}

func (m *MaxBot) commandMyID(_ context.Context, upd model.Update) (domain.Message, error) {
	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
		Text: fmt.Sprintf(
			"name: %s\nuser_id: %d\nchat_id: %d",
			upd.GetMessage().Sender.Name,
			upd.UserID,
			upd.ChatID,
		),
	}

	return msg, nil
}

func (m *MaxBot) stateCreateHandler(ctx context.Context, upd model.Update) (domain.Message, error) {
	msg := domain.Message{
		UserID: upd.UserID,
		Text:   "Запись создана",
	}

	_, err := m.repo.Add(ctx, domain.Todo{
		UserID:  upd.UserID,
		Message: upd.GetMessage().Body.Text,
	})
	if err != nil {
		return msg, fmt.Errorf("add: %w", err)
	}

	return msg, nil
}

func (m *MaxBot) callbackDeleteHandler(ctx context.Context, upd model.Update) error {
	pd := domain.CallBack{}
	err := pd.Value(upd.GetCallback().Payload)
	if err != nil {
		return fmt.Errorf("callback decode: %w", err)
	}

	td := domain.Todo{
		ID:     pd.InternalID,
		UserID: upd.UserID,
	}
	err = m.repo.Delete(ctx, td)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	msg := domain.Message{
		UserID: upd.UserID,
		ChatID: upd.ChatID,
	}

	records, err := m.repo.List(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	keyboard := model.NewKeyboard()
	for _, record := range records {
		txt := fmt.Sprintf("%s %s", record.Created.Format(time.DateTime), record.Message)
		pd = domain.CallBack{
			InternalID: record.ID,
			Key:        commandDelete,
		}
		keyboard.AddRow().AddCallBack(txt, pd.String())
	}

	if len(records) == 0 {
		return m.client.DeleteMessage(ctx, upd.GetMessage().Body.Mid)
	}

	msg.Keyboard = keyboard
	msg.Text = "Выберите запись для удаления:"

	err = m.client.EditMessage(ctx, upd.GetMessage().Body.Mid, msg)
	if err != nil {
		return fmt.Errorf("editMessage: %w", err)
	}

	return nil
}

func (m *MaxBot) callbackDefaultHandler(_ context.Context, _ model.Update) error {
	return nil
}

func cutLimit(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit]
}
