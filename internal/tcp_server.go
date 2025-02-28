package internal

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const Delim = '\n'
const DelimStr = string(Delim)

type iDB interface {
	Query(ctx context.Context, query string) (string, error)
}

type ServerTCP struct {
	cfg NetworkConfig

	db iDB

	isRunning bool
	logger    zerolog.Logger

	connections    int
	connectionsMtx sync.Mutex
}

func NewServerTCP(config NetworkConfig, db iDB, logger zerolog.Logger) *ServerTCP {
	return &ServerTCP{
		cfg:            config,
		db:             db,
		isRunning:      false,
		logger:         logger,
		connections:    0,
		connectionsMtx: sync.Mutex{},
	}
}

func (t *ServerTCP) Run(ctx context.Context) error {
	if t.isRunning {
		return errors.New("already running")
	}

	t.isRunning = true
	listener, err := net.ListenTCP("tcp", &t.cfg.Address)
	if err != nil {
		return err
	}

	defer func() {
		err := listener.Close()
		if err != nil {
			t.logger.Error().Err(err).Msg("error closing listener")
		} else {
			t.logger.Info().Msg("tcp listener closed")
		}
	}()

	t.logger.Info().Msg("listening on tcp://" + t.cfg.Address.String())

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		// Принимаем входящее соединение
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Ошибка при принятии соединения:", err)
			continue
		}

		// Обрабатываем соединение в отдельной горутине
		go t.handleConnection(ctx, conn)
	}
}

func (t *ServerTCP) handleConnection(ctx context.Context, conn net.Conn) {
	t.connectionsMtx.Lock()
	if t.connections+1 > t.cfg.MaxConnections {
		t.handleConnectionLimit(conn)
		t.connectionsMtx.Unlock()
		return
	}
	t.connections++
	t.connectionsMtx.Unlock()
	defer t.closeConnection(conn)

	t.logger.Info().Msgf("%s connected", conn.RemoteAddr())
	defer t.logger.Info().Msgf("%s diconnected", conn.RemoteAddr())

	// Чтение данных от клиента
	reader := bufio.NewReader(conn)
	for {
		done := make(chan struct{})
		var message string
		var err error
		go func() {
			defer close(done)
			message, err = reader.ReadString(Delim)
		}()

		select {
		case <-ctx.Done():
			return
		case <-done:
			if err = conn.SetDeadline(time.Now().Add(t.cfg.IdleTimeout)); err != nil {
				t.logger.Error().Err(err).Msg("error setting deadline")
				return
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				t.logger.Info().Msgf("client %s disconnected", conn.RemoteAddr())
				return
			}

			t.logger.Error().Err(err).Msg("error reading from tcp connection")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Убираем лишние пробелы и символы новой строки
		message = strings.TrimSpace(message)
		t.logger.Debug().Msgf("received message from %s: %s", conn.RemoteAddr(), message)

		// exec query
		response, err := t.db.Query(ctx, message)
		if err != nil {
			response = err.Error()
			t.logger.Error().Err(err).Msgf("error executing query %s", message)
		} else if response == "" {
			response = "ok"
		}

		response += DelimStr
		t.logger.Debug().Msgf("writing response '%s' to %s", response, conn.RemoteAddr())

		// Отправляем ответ клиенту
		_, err = conn.Write([]byte(response))
		if err != nil {
			t.logger.Err(err).Msg("on send response")
			return
		}

		t.logger.Debug().Msgf("wrote response to %s", conn.RemoteAddr())
	}
}

func (t *ServerTCP) handleConnectionLimit(conn net.Conn) {
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.logger.Error().Msgf("on set dedline for %s", conn.RemoteAddr())
		return
	}

	if _, err := conn.Write([]byte("too many connections" + DelimStr)); err != nil {
		t.logger.Error().Msgf("on send too many connections for %s", conn.RemoteAddr())
	}
}

func (t *ServerTCP) closeConnection(conn net.Conn) {
	t.connectionsMtx.Lock()
	t.connections--
	t.connectionsMtx.Unlock()

	err := conn.Close()
	if err != nil {
		t.logger.Error().Err(err).Msgf("error closing tcp connection %s", conn.RemoteAddr())
	} else {
		t.logger.Debug().Msgf("tcp connection %s closed", conn.RemoteAddr())
	}
}
