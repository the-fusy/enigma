package telegram

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"money/internal/entity"
	"money/internal/usecase"
	"strconv"
	"strings"
	"time"
)

type Bot struct {
	api                      *tgbotapi.BotAPI
	adminID                  int64
	idempotenceUsecase       *usecase.Idempotence
	createTransactionUsecase *usecase.CreateTransaction
	getTransactionsByDate    *usecase.GetTransactionsByDate
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

	return &Bot{
		api:                      botApi,
		adminID:                  adminID,
		idempotenceUsecase:       idempotenceUsecase,
		createTransactionUsecase: createTransactionUsecase,
		getTransactionsByDate:    getTransactionsByDate,
	}, nil
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

		if update.Message != nil {
			ok, err := b.idempotenceUsecase.Execute("telegram" + strconv.FormatInt(update.Message.Chat.ID, 10) + strconv.Itoa(update.Message.MessageID))
			if err != nil {
				fmt.Println(err)
				continue
			}

			if !ok {
				fmt.Println("Already handled", update.Message.Chat.ID, update.Message.MessageID)
				continue
			}

			if !update.Message.IsCommand() {
				continue
			}

			args := update.Message.CommandArguments()

			switch update.Message.Command() {
			case "create":
				if response, err := b.createTransaction(update.Message.Chat.ID, args); err != nil {
					b.handleError(update.Message, err)
				} else if response != nil {
					if _, err = b.api.Send(response); err != nil {
						fmt.Println(err)
					}
				}
			case "list":
				if response, err := b.listTransactions(update.Message.Chat.ID, args); err != nil {
					b.handleError(update.Message, err)
				} else if response != nil {
					if _, err = b.api.Send(response); err != nil {
						fmt.Println(err)
					}
				}
			}
		}
	}
}

func (b *Bot) createTransaction(user int64, message string) (tgbotapi.Chattable, error) {
	transaction, err := makeTransactionFromMessage(message)
	if err != nil {
		return nil, err
	}

	err = b.createTransactionUsecase.Execute(transaction)
	if err != nil {
		return nil, err
	}

	reply := tgbotapi.NewMessage(user, "Transaction created")
	return reply, nil
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

func (b *Bot) listTransactions(user int64, args string) (tgbotapi.Chattable, error) {
	var (
		date = time.Now().UTC()
		err  error
	)

	if args != "" {
		date, err = time.Parse("2006-01-02", args)
		if err != nil {
			return nil, fmt.Errorf("invalid date %s: %w", args, err)
		}
	}

	transactions, err := b.getTransactionsByDate.Execute(date)
	if err != nil {
		return nil, err
	}

	reply := tgbotapi.NewMessage(user, fmt.Sprintf("Transactions for %s: %v", date.Format("2006-01-02"), transactions))
	return reply, nil
}
