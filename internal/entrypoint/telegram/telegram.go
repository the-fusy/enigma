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
	api     *tgbotapi.BotAPI
	adminID int64

	idempotenceUsecase *usecase.Idempotence

	getUserStateUsecase  *usecase.GetUserstate
	saveUserStateUsecase *usecase.SaveUserstate

	createTransactionUsecase *usecase.CreateTransaction
	getTransactionsByDate    *usecase.GetTransactionsByDate
	getTransactionByID       *usecase.GetTransactionByID
}

func New(
	token string,
	adminID int64,
	idempotenceUsecase *usecase.Idempotence,
	getUserStateUsecase *usecase.GetUserstate,
	saveUserStateUsecase *usecase.SaveUserstate,
	createTransactionUsecase *usecase.CreateTransaction,
	getTransactionsByDate *usecase.GetTransactionsByDate,
	getTransactionByID *usecase.GetTransactionByID,
) (*Bot, error) {

	botApi, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		api:     botApi,
		adminID: adminID,

		idempotenceUsecase: idempotenceUsecase,

		getUserStateUsecase:  getUserStateUsecase,
		saveUserStateUsecase: saveUserStateUsecase,

		createTransactionUsecase: createTransactionUsecase,
		getTransactionsByDate:    getTransactionsByDate,
		getTransactionByID:       getTransactionByID,
	}

	b.fillStateNodes()

	return b, nil
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

		state, err := b.getUserStateUsecase.Execute(user.ID)
		if err != nil {
			b.handleError(update.Message, err)
			continue
		}

		if state.Name == "" {
			state.Name = entity.StartState
		}

		state.ChatID = user.ID
		if update.CallbackQuery != nil {
			state.MessageID = &update.CallbackQuery.Message.MessageID
		}

		state, err = stateNodes[state.Name].handleOut(state, update)
		if err != nil {
			b.handleError(update.Message, err)
			continue
		}

		err = b.saveUserStateUsecase.Execute(user.ID, state)
		if err != nil {
			b.handleError(update.Message, err)
			continue
		}

		reply, err := stateNodes[state.Name].handleIn(state)
		if err != nil {
			b.handleError(update.Message, err)
			continue
		}

		if reply != nil {
			_, err = b.api.Send(reply)
			if err != nil {
				b.handleError(update.Message, err)
				continue
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

func (b *Bot) fillStateNodes() {
	stateNodes[entity.StartState].handleIn = func(state entity.UserState) (tgbotapi.Chattable, error) {
		return tgbotapi.NewMessage(state.ChatID, "Welcome to Enigma"), nil
	}

	stateNodes[entity.CreateTransactionState].handleIn = b.createTransaction

	stateNodes[entity.ListTransactionsState].handleIn = b.listTransactions

	stateNodes[entity.ShowTransactionState].handleIn = b.showTransaction
}

func (b *Bot) createTransaction(state entity.UserState) (tgbotapi.Chattable, error) {
	transaction, err := makeTransactionFromArgs(state.Raw)
	if err != nil {
		return nil, err
	}

	err = b.createTransactionUsecase.Execute(transaction)
	if err != nil {
		return nil, err
	}

	return tgbotapi.NewMessage(state.ChatID, "Transaction created"), nil
}

func makeTransactionFromArgs(args string) (entity.Transaction, error) {
	messageParts := strings.SplitN(args, " ", 4)
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
	if message == nil {
		fmt.Println(err)
		return
	}

	_, err = b.api.Send(tgbotapi.NewMessage(message.Chat.ID, err.Error()))
	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bot) listTransactions(state entity.UserState) (tgbotapi.Chattable, error) {
	if state.Date == nil {
		return nil, errors.New("date is required")
	}

	transactions, err := b.getTransactionsByDate.Execute(*state.Date)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("Transactions for %s:\n\n", state.Date.Format("02.01.2006"))
	keyboard := newInlineKeyboard(5)

	if len(transactions) == 0 {
		message = "No transactions for " + state.Date.Format("02.01.2006")
	} else {
		for i, t := range transactions {
			message += fmt.Sprintf("%d. %s -> %s %v RUB: %s\n\n", i+1, t.FromAccount, t.ToAccount, t.Amount, t.Description)
			keyboard.addButton(strconv.Itoa(i+1), fmt.Sprintf("show %d", t.ID))
		}
		keyboard.fillLastRowWithEmptyButtons()

		message += "Choose one to edit"
	}

	keyboard.addButton("⬅️", fmt.Sprintf("list %s", state.Date.AddDate(0, 0, -1).Format("02.01.2006")))
	keyboard.addButton("➡️", fmt.Sprintf("list %s", state.Date.AddDate(0, 0, 1).Format("02.01.2006")))

	if state.MessageID != nil {
		reply := tgbotapi.NewEditMessageText(state.ChatID, *state.MessageID, message)
		reply.ReplyMarkup = keyboard.markup()
		return reply, nil
	}

	reply := tgbotapi.NewMessage(state.ChatID, message)
	reply.ReplyMarkup = keyboard.markup()
	return reply, nil
}

func (b *Bot) showTransaction(state entity.UserState) (tgbotapi.Chattable, error) {
	if state.TransactionID == nil {
		return nil, errors.New("transaction id is required")
	}

	transactionID := *state.TransactionID

	transaction, err := b.getTransactionByID.Execute(transactionID)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("Transaction #%d:\n\n", transaction.ID)
	message += fmt.Sprintf("Date: %s\n", transaction.Date.Format("02.01.2006"))
	message += fmt.Sprintf("From: %s\n", transaction.FromAccount)
	message += fmt.Sprintf("To: %s\n", transaction.ToAccount)
	message += fmt.Sprintf("Amount: %v RUB\n", transaction.Amount)
	message += fmt.Sprintf("Description: %s", transaction.Description)

	keyboard := newInlineKeyboard(3)
	for _, t := range []string{"Date", "From", "To", "Amount", "Description"} {
		keyboard.addButton(t, fmt.Sprintf("edit %s %d", strings.ToLower(t), transaction.ID))
	}

	keyboard.addRow()
	keyboard.addButton("↩", fmt.Sprintf("list %s", transaction.Date.Format("02.01.2006")))

	if state.MessageID != nil {
		reply := tgbotapi.NewEditMessageText(state.ChatID, *state.MessageID, message)
		reply.ReplyMarkup = keyboard.markup()
		return reply, nil
	}

	reply := tgbotapi.NewMessage(state.ChatID, message)
	reply.ReplyMarkup = keyboard.markup()
	return reply, nil
}
