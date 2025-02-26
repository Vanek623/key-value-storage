package main

import (
	"bufio"
	"context"
	"github.com/rs/zerolog"
	"key-value-storage/internal"
	"os"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	logger := zerolog.New(os.Stdout)
	db := internal.NewDB(internal.NewParser(logger), internal.NewStorage(internal.NewEngine(), logger), logger)
	ctx := context.Background()

	logger.Println("Вводите запросы к БД или напишите 'exit' для завершение работы")
	for {
		logger.Print("Введите строку: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			logger.Println("Ошибка при чтении ввода:", err)
			continue
		}

		// Убираем символ новой строки из ввода
		input = input[:len(input)-1]

		// Пример условия для выхода из цикла
		if input == "exit" {
			logger.Println("Выход из программы...")
			break
		}

		resp, err := db.Query(ctx, input)
		if err != nil {
			logger.Println("Ошибка обработки запроса:", err)
			continue
		}

		logger.Println("Результат запроса:", resp)
	}
}
