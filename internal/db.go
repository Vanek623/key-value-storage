package internal

import (
	"context"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type iParser interface {
	Parse(string) (Command, error)
}

type iStorage interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) (string, error)
	Del(context.Context, string) error
}

type DB struct {
	parser  iParser
	storage iStorage
	logger  zerolog.Logger
}

func NewDB(parser iParser, storage iStorage, logger zerolog.Logger) *DB {
	return &DB{
		parser:  parser,
		storage: storage,
		logger:  logger,
	}
}

func (db *DB) Query(ctx context.Context, query string) (string, error) {
	command, err := db.parser.Parse(query)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse command")
	}

	var resp string
	switch command.Type {
	case Get:
		resp, err = db.storage.Get(ctx, command.Args[0])
		if err != nil {
			return "", errors.Wrap(err, "failed to get value")
		}
	case Set:
		err = db.storage.Set(ctx, command.Args[0], command.Args[1])
		if err != nil {
			return "", errors.Wrap(err, "failed to set value")
		}
	case Del:
		err = db.storage.Del(ctx, command.Args[0])
		if err != nil {
			return "", errors.Wrap(err, "failed to delete value")
		}
	}

	return resp, nil
}
