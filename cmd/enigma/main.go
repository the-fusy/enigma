package main

import (
	"context"
	"flag"
	"log"

	"enigma/internal/entrypoint/telegram"
	"enigma/internal/usecase"
	"enigma/internal/usecase/repository/idempotence"
	"enigma/internal/usecase/repository/transaction"

	bolt "go.etcd.io/bbolt"
)

var token = flag.String("token", "", "telegram bot token")
var adminID = flag.Int64("admin", 0, "admin's telegram id")

func main() {
	flag.Parse()

	if *token == "" {
		log.Fatalln("[ERROR] -token argument is required")
		return
	}

	if *adminID == 0 {
		log.Fatalln("[ERROR] -admin argument is required")
		return
	}

	db, err := bolt.Open("test.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	idempotenceRepository, err := idempotence.NewBoltDB(db)
	if err != nil {
		log.Fatal(err)
	}
	idempotenceUsecase := usecase.NewIdempotence(idempotenceRepository)

	transactionRepository, err := transaction.NewBoltDB(db)
	if err != nil {
		log.Fatal(err)
	}
	createTransactionUsecase := usecase.NewCreateTransaction(transactionRepository)
	getTransactionsByDateUsecase := usecase.NewGetTransactionsByDate(transactionRepository)

	bot, err := telegram.New(*token, *adminID, idempotenceUsecase, createTransactionUsecase, getTransactionsByDateUsecase)
	if err != nil {
		log.Fatal(err)
	}

	bot.Start(context.Background())

	select {}
}