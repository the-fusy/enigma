package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type reply struct {
	text           string
	inlineKeyboard *tgbotapi.InlineKeyboardMarkup
}
