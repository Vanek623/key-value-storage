package internal

import (
	"github.com/pkg/errors"
	"sync"
)

type InMemoryEngine struct {
	m   map[string]string
	mtx sync.RWMutex
}

func NewInMemoryEngine() *InMemoryEngine {
	return &InMemoryEngine{
		m:   make(map[string]string),
		mtx: sync.RWMutex{},
	}
}

func NewEngine(engineType EngineType) (iEngine, error) {
	switch engineType {
	case InMemoryEngineType:
		return NewInMemoryEngine(), nil
	default:
		return nil, errors.New("invalid engine type")
	}
}

func (e *InMemoryEngine) Get(key string) (string, bool) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	val, has := e.m[key]

	return val, has
}

func (e *InMemoryEngine) Set(key string, value string) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.m[key] = value
}

func (e *InMemoryEngine) Del(key string) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	delete(e.m, key)
}
