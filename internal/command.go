package internal

import (
	"github.com/pkg/errors"
)

type CommandType string

const (
	Get CommandType = "GET"
	Set CommandType = "SET"
	Del CommandType = "DEL"
)

type Command struct {
	Type CommandType
	Args []string
}

func (c Command) validate() error {
	var msg string
	switch c.Type {
	case Get, Del:
		if len(c.Args) != 1 {
			msg = "args count must be 1"
		}
	case Set:
		if len(c.Args) != 2 {
			msg = "args count must be 2"
		}
	}

	if msg != "" {
		return errors.Wrap(ErrInvalidCommand, msg)
	}

	return nil
}
