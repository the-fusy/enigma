package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type inlineKeyboard struct {
	rows             [][]tgbotapi.InlineKeyboardButton
	maxButtonsPerRow int
}

func newInlineKeyboard(maxButtonsPerRow int) *inlineKeyboard {
	return &inlineKeyboard{
		rows:             make([][]tgbotapi.InlineKeyboardButton, 0),
		maxButtonsPerRow: maxButtonsPerRow,
	}
}

func (k *inlineKeyboard) addButton(text, data string) {
	if len(k.rows) == 0 {
		k.addRow()
	}

	lastRowIndex := len(k.rows) - 1
	if len(k.rows[lastRowIndex]) == k.maxButtonsPerRow-1 {
		k.addRow()
	}

	k.rows[lastRowIndex] = append(k.rows[lastRowIndex], tgbotapi.NewInlineKeyboardButtonData(text, data))
}

func (k *inlineKeyboard) addRow() {
	k.rows = append(k.rows, []tgbotapi.InlineKeyboardButton{})
}

func (k *inlineKeyboard) fillLastRowWithEmptyButtons() {
	if len(k.rows) == 0 {
		return
	}

	rowsCount := len(k.rows)
	for len(k.rows) == rowsCount {
		k.addButton(" ", "empty")
	}
}

func (k *inlineKeyboard) markup() *tgbotapi.InlineKeyboardMarkup {
	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: k.rows,
	}
}
