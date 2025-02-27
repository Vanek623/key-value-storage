package client

import (
	"bufio"
	"context"
	"github.com/rs/zerolog"
	"key-value-storage/internal"
	"net"
	"sync"
	"time"
)

type TCP struct {
	conn        net.Conn
	logger      zerolog.Logger
	readTimeout time.Duration
}

func NewClientTCP(address string, logger zerolog.Logger, readTimeout time.Duration) (c *TCP, cl func(), err error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	logger.Debug().Msgf("connected to %s", address)

	cl = func() {
		err := conn.Close()
		if err != nil {
			logger.Err(err).Msg("error closing connection")
		}

		logger.Info().Msg("closed connection")
	}

	return &TCP{
		conn:        conn,
		logger:      logger,
		readTimeout: readTimeout,
	}, cl, nil
}

func (c TCP) Query(ctx context.Context, query string) (string, error) {
	var result string
	var err error

	done := make(chan struct{})
	go func() {
		defer close(done)

		query += internal.DelimStr
		_, err = c.conn.Write([]byte(query))
		if err != nil {
			return
		}

		c.logger.Debug().Msgf("sent query: %s", query)

		reader := bufio.NewReader(c.conn)
		// Читаем ответ от сервера
		result, err = reader.ReadString(internal.Delim)
		if err != nil {
			return
		}

		c.logger.Debug().Msgf("received response: %s", result)
	}()
	ctx, cl := context.WithTimeout(ctx, c.readTimeout)
	defer cl()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-done:
	}

	return result, err
}

type GroupTCP struct {
	clients []*TCP
}

func NewGroupTCP(address string, logger zerolog.Logger, readTimeout time.Duration, count int) (c *GroupTCP, cl func(), err error) {
	cls := make([]func(), 0, count)
	clients := make([]*TCP, 0, count)
	for range count {
		client, cl, err := NewClientTCP(address, logger, readTimeout)
		if err != nil {
			for _, cl := range cls {
				cl()
			}

			return nil, nil, err
		}

		cls = append(cls, cl)
		clients = append(clients, client)
	}

	cl = func() {
		for _, cl := range cls {
			cl()
		}
	}

	return &GroupTCP{clients: clients}, cl, nil
}

func (c GroupTCP) Query(ctx context.Context, query string) (string, error) {
	type resp struct {
		result string
		err    error
	}

	done := make(chan resp, len(c.clients))
	wg := sync.WaitGroup{}
	for _, client := range c.clients {
		go func() {
			wg.Add(1)
			defer wg.Done()

			r, err := client.Query(ctx, query)
			done <- resp{
				result: r,
				err:    err,
			}
		}()
	}

	wg.Wait()

	var lastErr error
	for r := range done {
		if r.err == nil {
			return r.result, nil
		}

		lastErr = r.err
	}

	return "", lastErr
}
