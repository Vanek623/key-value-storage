package internal

import "sync"

type Engine struct {
	m   map[string]string
	mtx sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		m:   make(map[string]string),
		mtx: sync.RWMutex{},
	}
}

func (e *Engine) Get(key string) (string, bool) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	val, has := e.m[key]

	return val, has
}

func (e *Engine) Set(key string, value string) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.m[key] = value
}

func (e *Engine) Del(key string) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	delete(e.m, key)
}
