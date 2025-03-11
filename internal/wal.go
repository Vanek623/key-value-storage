package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"github.com/golang/groupcache/singleflight"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"io"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	newBatchSfGroup = "new batch"

	commandBufferSize = 1024
)

type Writer struct {
	maxSize       int64
	size          int64
	segment       *os.File
	segmentWriter *bufio.Writer

	encoder *gob.Encoder
	buffer  *bytes.Buffer

	dir string
}

func NewWriter(dirPath string, maxSize int64) (*Writer, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, commandBufferSize))
	w := &Writer{
		maxSize:       maxSize,
		size:          0,
		segment:       nil,
		segmentWriter: nil,
		encoder:       gob.NewEncoder(buffer),
		buffer:        buffer,
		dir:           dirPath,
	}

	err := w.openSegment(dirPath)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Writer) Write(commands []Command) error {
	for cmd := range commands {
		err := w.encoder.Encode(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to encode command")
		}

		l := int64(w.buffer.Len())
		if l+w.size > w.maxSize {
			if err = w.segmentWriter.Flush(); err != nil {
				return errors.Wrap(err, "failed to flush segment")
			}

			if err = w.nextSegment(); err != nil {
				return errors.Wrap(err, "failed to open next segment")
			}
		}

		if _, err = w.segmentWriter.Write(w.buffer.Bytes()); err != nil {
			return errors.Wrap(err, "failed to write to segment")
		}

		w.size += l
	}

	if err := w.segmentWriter.Flush(); err != nil {
		return errors.Wrap(err, "failed to flush segment")
	}

	return nil
}

func (w *Writer) openSegment(dirPath string) error {
	if w.segment != nil {
		return errors.New("segment already open")
	}

	err := os.MkdirAll(dirPath, 0750)
	if err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "failed to create dirs %s", dirPath)
	}

	dir, err := os.Open(dirPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open dir %s", dirPath)
	}

	currentFileName, size, err := w.findFileName(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to get segment name")
	}

	segment, err := os.OpenFile(path.Join(dirPath, currentFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.Wrapf(err, "failed to open segment %s", currentFileName)
	}

	w.segmentWriter = bufio.NewWriter(segment)
	w.size = size
	w.segment = segment

	return nil
}

func (w *Writer) nextSegment() error {
	segmentNum, err := strconv.Atoi(w.segment.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to read segment num")
	}

	nextSegmentName := strconv.Itoa(segmentNum + 1)
	nextSegment, err := os.OpenFile(path.Join(w.dir, nextSegmentName), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrapf(err, "failed to open next segment %s", nextSegmentName)
	}
	info, err := nextSegment.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to read next segment info")
	}
	if info.Size() != 0 {
		return errors.New("next segment isn't empty")
	}

	if err = w.segment.Close(); err != nil {
		return errors.Wrapf(err, "failed to close old segment %s", w.segment.Name())
	}

	w.segment = nextSegment
	w.size = 0
	w.segmentWriter.Reset(nextSegment)

	return nil
}

// ищет/создает имя последнего сегмента и его размер
func (w *Writer) findFileName(dir *os.File) (string, int64, error) {
	files, err := dir.ReadDir(-1)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "1", 0, nil
		}
		return "", 0, errors.Wrapf(err, "failed to read dir %s", dir.Name())
	}

	files = slices.DeleteFunc(files, func(e os.DirEntry) bool {
		if e.IsDir() {
			return true
		}

		isInvalidName := strings.ContainsFunc(e.Name(), func(r rune) bool {
			return r < '0' || r > '9'
		})
		return isInvalidName
	})
	slices.SortFunc(files, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	maxFile := files[len(files)-1]
	inf, err := maxFile.Info()
	if err != nil {
		return "", 0, errors.Wrapf(err, "failed to read max file info")
	}

	currentFileName := maxFile.Name()
	if size := inf.Size(); size < w.maxSize {
		return currentFileName, size, nil
	}

	segNum, err := strconv.Atoi(currentFileName)
	if err != nil {
		return "", 0, errors.Wrapf(err, "failed to read segment num from filename: %s", currentFileName)
	}
	segNum++
	currentFileName = strconv.Itoa(segNum)

	return currentFileName, 0, nil
}

type Wal struct {
	cfg WalConfig
	t   *time.Ticker

	batchMtx sync.Mutex
	batch    *Batch
	writer   *Writer

	logger zerolog.Logger

	sf singleflight.Group
}

func NewWal(cfg WalConfig, logger zerolog.Logger) (*Wal, error) {
	segmentWriter, err := NewWriter(cfg.DataDir, int64(cfg.SegmentSize))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create segment writer")
	}

	w := &Wal{
		cfg:      cfg,
		t:        time.NewTicker(cfg.BatchTimeout),
		batchMtx: sync.Mutex{},
		batch:    NewBatch(cfg.BatchSize),
		writer:   segmentWriter,
		logger:   logger,
		sf:       singleflight.Group{},
	}

	return w, nil
}

func (w *Wal) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.t.C:
			if err := w.flush(); err != nil {
				w.logger.Error().Err(err).Msg("failed to flush")
			}
		}
	}
}

func (w *Wal) Push(ctx context.Context, cmd Command) error {
	if !w.cfg.Enabled {
		return nil
	}

	w.batchMtx.Lock()
	batch := w.batch
	batch.data = append(batch.data, cmd)
	l := len(w.batch.data)
	w.batchMtx.Unlock()

	if l > w.cfg.BatchSize {
		go func() {
			if err := w.flush(); err != nil {
				w.logger.Error().Err(err).Msg("failed to flush")
			}
		}()
	}

	select {
	case <-batch.flushDoneCh:
		return batch.flushErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Wal) flush() error {
	_, err := w.sf.Do(newBatchSfGroup, func() (interface{}, error) {
		w.logger.Debug().Msgf("batch is full, flushing...")

		w.batchMtx.Lock()
		batch := w.batch
		w.batch = NewBatch(w.cfg.BatchSize)
		w.batchMtx.Unlock()

		defer close(batch.flushDoneCh)

		err := w.writer.Write(batch.data)
		if err != nil {
			batch.flushErr = err
			return nil, err
		}

		return nil, nil
	})

	return err
}

type Batch struct {
	flushErr    error
	flushDoneCh chan struct{}
	data        []Command
}

func NewBatch(size int) *Batch {
	return &Batch{
		flushErr:    nil,
		flushDoneCh: make(chan struct{}),
		data:        make([]Command, 0, size),
	}
}
