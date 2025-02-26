package internal

import (
	"context"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var ErrNotFound = errors.New("key not found")

type iEngine interface {
	Set(key string, value string)
	Get(key string) (string, bool)
	Del(key string)
}

type Storage struct {
	engine iEngine
	logger zerolog.Logger
}

func NewStorage(engine iEngine, logger zerolog.Logger) *Storage {
	return &Storage{
		engine: engine,
		logger: logger,
	}
}

func (s *Storage) Set(_ context.Context, key string, value string) error {
	s.engine.Set(key, value)

	return nil
}

func (s *Storage) Get(_ context.Context, key string) (string, error) {
	val, has := s.engine.Get(key)
	if !has {
		return "", ErrNotFound
	}
	return val, nil
}

func (s *Storage) Del(_ context.Context, key string) error {
	s.engine.Del(key)

	return nil
}
