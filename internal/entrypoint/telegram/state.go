package telegram

import (
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"enigma/internal/entity"
)

type stateNode struct {
	stateName string

	handleIn  func(state entity.UserState) (tgbotapi.Chattable, error)
	handleOut func(state entity.UserState, update tgbotapi.Update) (entity.UserState, error)

	transitionByText     transition
	transitionByCommand  map[string]transition
	transitionByCallback map[string]transition
}

type transition struct {
	node   *stateNode
	parser argsParser
}

func (n *stateNode) addTransitionByText(next *stateNode, parser argsParser) {
	n.transitionByText = transition{next, parser}
}

func (n *stateNode) addTransitionByCommand(command string, next *stateNode, parser argsParser) {
	if n.transitionByCommand == nil {
		n.transitionByCommand = make(map[string]transition)
	}
	n.transitionByCommand[command] = transition{next, parser}
}

func (n *stateNode) addTransitionByCallback(query string, next *stateNode, parser argsParser) {
	if n.transitionByCallback == nil {
		n.transitionByCallback = make(map[string]transition)
	}
	n.transitionByCallback[query] = transition{next, parser}
}

func (n *stateNode) getTransitionWithArgs(update tgbotapi.Update) (transition, string, error) {
	var t transition
	var args string

	if update.Message != nil {
		if update.Message.IsCommand() {
			t = n.transitionByCommand[update.Message.Command()]
			args = update.Message.CommandArguments()
		} else {
			t = n.transitionByText
			args = update.Message.Text
		}
	} else if update.CallbackQuery != nil {
		split := strings.SplitN(update.CallbackQuery.Data, " ", 2)
		command := split[0]
		if len(split) > 1 {
			args = split[1]
		}
		t = n.transitionByCallback[command]
	}

	if t.node == nil {
		return transition{}, "", errors.New("no transition")
	}

	return t, args, nil
}

func (n *stateNode) getNextState(state entity.UserState, update tgbotapi.Update) (entity.UserState, error) {
	t, args, err := n.getTransitionWithArgs(update)
	if err != nil {
		return entity.UserState{}, err
	}

	if t.parser != nil {
		state, err = t.parser(state, args)
		if err != nil {
			return entity.UserState{}, err
		}
	}

	state.Name = t.node.stateName

	return state, nil
}

var stateNodes = make(map[string]*stateNode)

func init() {
	for _, stateName := range []string{
		entity.StartState,
		entity.CreateTransactionState,
		entity.ShowTransactionState,
		entity.ListTransactionsState,
	} {
		if _, ok := stateNodes[stateName]; !ok {
			stateNodes[stateName] = &stateNode{
				stateName: stateName,
			}
			stateNodes[stateName].handleOut = stateNodes[stateName].getNextState
		}
	}

	stateNodes[entity.StartState].addTransitionByCommand("start", stateNodes[entity.StartState], nil)
	stateNodes[entity.StartState].addTransitionByCommand("create", stateNodes[entity.CreateTransactionState], nil)
	stateNodes[entity.StartState].addTransitionByCommand("list", stateNodes[entity.ListTransactionsState], dateParser)

	stateNodes[entity.ListTransactionsState].addTransitionByCommand("start", stateNodes[entity.StartState], nil)
	stateNodes[entity.ListTransactionsState].addTransitionByCommand("create", stateNodes[entity.CreateTransactionState], nil)
	stateNodes[entity.ListTransactionsState].addTransitionByCommand("list", stateNodes[entity.ListTransactionsState], dateParser)
	stateNodes[entity.ListTransactionsState].addTransitionByCallback("list", stateNodes[entity.ListTransactionsState], dateParser)
	stateNodes[entity.ListTransactionsState].addTransitionByCallback("show", stateNodes[entity.ShowTransactionState], transactionIDParser)

	stateNodes[entity.ShowTransactionState].addTransitionByCommand("start", stateNodes[entity.StartState], nil)
	stateNodes[entity.ShowTransactionState].addTransitionByCommand("create", stateNodes[entity.CreateTransactionState], nil)
	stateNodes[entity.ShowTransactionState].addTransitionByCommand("list", stateNodes[entity.ListTransactionsState], dateParser)
	stateNodes[entity.ShowTransactionState].addTransitionByCallback("list", stateNodes[entity.ListTransactionsState], dateParser)
}
