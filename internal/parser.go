package internal

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"slices"
	"strings"
	"unicode/utf8"
)

// query = set_command | get_command | del_command
//
//set_command = "SET" argument argument
//get_command = "GET" argument
//del_command = "DEL" argument
//argument    = punctuation | letter | digit { punctuation | letter | digit }
//
//punctuation = "*" | "/" | "_" | ...
//letter      = "a" | ... | "z" | "A" | ... | "Z"
//digit       = "0" | ... | "9"
//

var ErrInvalidCommand = errors.New("invalid command")

type Parser struct {
	logger zerolog.Logger
}

func NewParser(logger zerolog.Logger) Parser {
	return Parser{logger: logger}
}

func (p Parser) Parse(line string) (Command, error) {
	p.logger.Debug().Msgf("parsing %s", line)
	tokens := strings.Split(line, " ")
	for i, param := range tokens {
		invalidCharIndex := strings.IndexFunc(param, func(r rune) bool {
			return !isValidChar(r)
		})
		if invalidCharIndex != -1 {
			r, _ := utf8.DecodeRuneInString(param[invalidCharIndex:])
			return Command{}, errors.Wrapf(ErrInvalidCommand, "invalid char %s[%d]", string(r), invalidCharIndex)
		}

		tokens[i] = strings.TrimSpace(tokens[i])
	}

	tokens = slices.DeleteFunc(tokens, func(s string) bool {
		return s == ""
	})

	if len(tokens) == 0 {
		return Command{}, errors.Wrapf(ErrInvalidCommand, "invalid command len %d", len(tokens))
	}

	log.Debug().Msgf("tokens: %v", tokens)

	commandType := CommandType(tokens[0])
	switch commandType {
	case Get, Set, Del:
	default:
		return Command{}, errors.Wrapf(ErrInvalidCommand, "invalid command type %s", tokens[0])
	}

	c := Command{
		Type: commandType,
		Args: tokens[1:],
	}
	if err := c.validate(); err != nil {
		return Command{}, err
	}

	return c, nil
}

func isValidChar(char rune) bool {
	if char >= '0' && char <= '9' {
		return true
	}

	if char >= 'a' && char <= 'z' {
		return true
	}

	if char >= 'A' && char <= 'Z' {
		return true
	}

	if char == '*' || char == '_' || char == '/' {
		return true
	}

	return false
}
