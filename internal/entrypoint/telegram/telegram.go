package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"enigma/internal/entity"
	"enigma/internal/usecase"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api                      *tgbotapi.BotAPI
	adminID                  int64
	idempotenceUsecase       *usecase.Idempotence
	createTransactionUsecase *usecase.CreateTransaction
	getTransactionsByDate    *usecase.GetTransactionsByDate

	commands map[string]func(args string) (*reply, error)
}

func New(
	token string,
	adminID int64,
	idempotenceUsecase *usecase.Idempotence,
	createTransactionUsecase *usecase.CreateTransaction,
	getTransactionsByDate *usecase.GetTransactionsByDate,
) (*Bot, error) {

	botApi, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		api:                      botApi,
		adminID:                  adminID,
		idempotenceUsecase:       idempotenceUsecase,
		createTransactionUsecase: createTransactionUsecase,
		getTransactionsByDate:    getTransactionsByDate,

		commands: make(map[string]func(args string) (*reply, error)),
	}

	b.Register("create", b.createTransaction)
	b.Register("list", b.listTransactions)

	return b, nil
}

func (b *Bot) Register(command string, handler func(args string) (*reply, error)) {
	b.commands[command] = handler
}

func (b *Bot) Start(ctx context.Context) {
	config := tgbotapi.NewUpdate(0)
	config.Timeout = 60

	updates := b.api.GetUpdatesChan(config)
	go b.HandleUpdates(ctx, updates)
}

func (b *Bot) HandleUpdates(_ context.Context, updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		user := update.SentFrom()
		if user.ID != b.adminID {
			continue
		}

		if ok, err := b.checkIfFirstHandle(update); err != nil {
			fmt.Println(err)
			continue
		} else if !ok {
			continue
		}

		if update.Message != nil {
			if !update.Message.IsCommand() {
				continue
			}

			handler, ok := b.commands[update.Message.Command()]
			if !ok {
				continue
			}

			reply, err := handler(update.Message.CommandArguments())
			if err != nil {
				b.handleError(update.Message, err)
				continue
			}

			message := tgbotapi.NewMessage(update.Message.Chat.ID, reply.text)
			message.ReplyMarkup = reply.inlineKeyboard

			_, err = b.api.Send(message)
			if err != nil {
				fmt.Println(err)
			}
		}

		if update.CallbackQuery != nil {
			ca := strings.SplitN(update.CallbackQuery.Data, " ", 2)
			if len(ca) != 2 {
				continue
			}

			handler, ok := b.commands[ca[0]]
			if !ok {
				continue
			}

			reply, err := handler(ca[1])
			if err != nil {
				b.handleError(update.CallbackQuery.Message, err)
				continue
			}

			message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, reply.text)
			message.ReplyMarkup = reply.inlineKeyboard

			_, err = b.api.Send(message)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (b *Bot) checkIfFirstHandle(update tgbotapi.Update) (bool, error) {
	id := "telegram"
	if update.Message != nil {
		id += strconv.FormatInt(update.Message.Chat.ID, 10) + strconv.Itoa(update.Message.MessageID)
	} else if update.CallbackQuery != nil {
		id += strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10) + update.CallbackQuery.ID
	}
	return b.idempotenceUsecase.Execute(id)
}

func (b *Bot) createTransaction(message string) (*reply, error) {
	transaction, err := makeTransactionFromMessage(message)
	if err != nil {
		return nil, err
	}

	err = b.createTransactionUsecase.Execute(transaction)
	if err != nil {
		return nil, err
	}

	return &reply{text: "Transaction created"}, nil
}

func makeTransactionFromMessage(message string) (entity.Transaction, error) {
	messageParts := strings.SplitN(message, " ", 4)
	if len(messageParts) != 4 {
		return entity.Transaction{}, errors.New("invalid message format")
	}

	amount, err := strconv.ParseFloat(messageParts[2], 64)
	if err != nil {
		return entity.Transaction{}, fmt.Errorf("invalid amount %s: %w", messageParts[2], err)
	}

	transaction := entity.Transaction{
		Date:        time.Now().UTC(),
		FromAccount: messageParts[0],
		ToAccount:   messageParts[1],
		Amount:      amount,
		Description: messageParts[3],
	}

	return transaction, nil
}

func (b *Bot) handleError(message *tgbotapi.Message, err error) {
	_, err = b.api.Send(tgbotapi.NewMessage(message.Chat.ID, err.Error()))
	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bot) listTransactions(args string) (*reply, error) {
	var (
		date = time.Now().UTC()
		err  error
	)

	if args != "" {
		date, err = time.Parse("02.01.2006", args)
		if err != nil {
			return nil, fmt.Errorf("invalid date %s: %w", args, err)
		}
	}

	transactions, err := b.getTransactionsByDate.Execute(date)
	if err != nil {
		return nil, err
	}

	r := reply{
		text: fmt.Sprintf("Transactions for %s:\n\n", date.Format("02.01.2006")),
	}

	for i, t := range transactions {
		r.text += fmt.Sprintf("%d. %s -> %s %v RUB: %s /edit_%d\n\n", i+1, t.FromAccount, t.ToAccount, t.Amount, t.Description, t.ID)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️", fmt.Sprintf("list %s", date.AddDate(0, 0, -1).Format("02.01.2006"))),
		tgbotapi.NewInlineKeyboardButtonData("➡️", fmt.Sprintf("list %s", date.AddDate(0, 0, 1).Format("02.01.2006"))),
	))
	r.inlineKeyboard = &keyboard

	return &r, nil
}
