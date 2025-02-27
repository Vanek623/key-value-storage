package internal

import (
	"bufio"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"os"
)

type Console struct {
	db     iDB
	logger zerolog.Logger
}

func NewConsole(db iDB, logger zerolog.Logger) *Console {
	return &Console{db: db, logger: logger}
}

func (c *Console) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	c.logger.Info().Msg("run console mode")
	fmt.Println("Вводите запросы к БД или напишите 'exit' для завершение работы")
	defer func() {
		c.logger.Info().Msg("stop console mode")
		fmt.Println("Завершение работы")
	}()
	for {
		fmt.Println("Введите строку: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			c.logger.Err(err).Msg("on read input")
			fmt.Println("Ошибка при чтении ввода: ", err)
			continue
		}
		c.logger.Debug().Msg("Input: " + input)

		// Убираем символ новой строки из ввода
		input = input[:len(input)-1]

		const exitCmd = "exit"
		if input == exitCmd {
			break
		}

		resp, err := c.db.Query(ctx, input)
		if err != nil {
			c.logger.Err(err).Msg("on exec query")
			fmt.Println("Ошибка выполнения запроса: " + err.Error())
			continue
		}

		fmt.Println("Результат запроса: ", resp)
	}

	return nil
}
