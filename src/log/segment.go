package log

import (
	"fmt"
	"os"
	"path"

	api "github.com/gcleroux/projet-A23/api/v1"
	"google.golang.org/protobuf/proto"
)

// A segment keeps track of the storeFile and the indexFile for a log with the relative offset
type segment struct {
	store                  *store
	index                  *index
	baseOffset, nextOffset uint64
	config                 LogConfig
}

func newSegment(dir string, baseOffset uint64, c LogConfig) (*segment, error) {
	s := &segment{
		baseOffset: baseOffset,
		config:     c,
	}

	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}

	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")),
		os.O_RDWR|os.O_CREATE,
		0o644,
	)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}

	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		// Creating a segment with a non-empty indexFile
		s.nextOffset = baseOffset + uint64(off) + 1
	}
	return s, nil
}

func (s *segment) Append(record *api.Record) (offset uint64, err error) {
	currentOffset := s.nextOffset
	record.Offset = currentOffset

	p, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}

	_, pos, err := s.store.Append(p)
	if err != nil {
		return 0, err
	}

	err = s.index.Write(
		// index offsets are relative to base offset
		uint32(s.nextOffset-uint64(s.baseOffset)),
		pos,
	)
	if err != nil {
		return 0, err
	}
	s.nextOffset++
	return currentOffset, nil
}

func (s *segment) Read(offset uint64) (*api.Record, error) {
	_, pos, err := s.index.Read(int64(offset - s.baseOffset))
	if err != nil {
		return nil, err
	}

	data, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}

	record := &api.Record{}
	err = proto.Unmarshal(data, record)
	return record, err
}

func (s *segment) IsMaxed() bool {
	return s.store.size >= s.config.GetSegment().MaxStoreBytes ||
		s.index.size >= s.config.GetSegment().MaxIndexBytes
}

func (s *segment) Close() error {
	if err := s.index.Close(); err != nil {
		return err
	}
	if err := s.store.Close(); err != nil {
		return err
	}
	return nil
}

func (s *segment) Remove() error {
	if err := s.Close(); err != nil {
		return err
	}
	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}
	if err := os.Remove(s.store.Name()); err != nil {
		return err
	}
	return nil
}
