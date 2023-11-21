package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

// Encoding used for record sizes
var encoding = binary.BigEndian

// Number of bytes used to store the record's length
const lenWidth = 8

// The store writes to buffered IO instead of the file directly to improve perf and reduce syscalls
// This way many small logs will be buffered and written all at once
type store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

// newStore creates a new log store based on the provided file.
func newStore(f *os.File) (*store, error) {
	fileInfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	size := uint64(fileInfo.Size())
	return &store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}, nil
}

// Append adds a new record to the log store with the given data and returns
// the number of bytes written, position, and any encountered error.
func (s *store) Append(data []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos = s.size

	if err := binary.Write(s.buf, encoding, uint64(len(data))); err != nil {
		return 0, 0, err
	}

	w, err := s.buf.Write(data)
	if err != nil {
		return 0, 0, err
	}

	w += lenWidth
	s.size += uint64(w)
	return uint64(w), pos, nil
}

// Read retrieves a record from the log store at the specified position.
func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flushing the buffer in case the data we want to read hasn't been written to disk yet
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	recordSize := make([]byte, lenWidth)
	if _, err := s.File.ReadAt(recordSize, int64(pos)); err != nil {
		return nil, err
	}

	record := make([]byte, encoding.Uint64(recordSize))
	if _, err := s.File.ReadAt(record, int64(pos+lenWidth)); err != nil {
		return nil, err
	}
	return record, nil
}

// ReadAt reads data from the log store at the specified offset and returns the number
// of bytes read and any encountered error.
func (s *store) ReadAt(data []byte, offset int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flushing the buffer in case the data we want to read hasn't been written to disk yet
	if err := s.buf.Flush(); err != nil {
		return 0, err
	}
	return s.File.ReadAt(data, offset)
}

// Close closes the log store, ensuring proper flushing of the buffer and closing of the file.
func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return err
	}
	return s.File.Close()
}
