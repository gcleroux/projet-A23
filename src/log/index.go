package log

import (
	"io"
	"os"

	"github.com/edsrzf/mmap-go"
)

const (
	offsetWidth uint64 = 4
	posWidth    uint64 = 8
	entryWidth  uint64 = offsetWidth + posWidth
)

// index defines the index file used to retrieve the logs inside the store.
// To retrieve the log's data, we need the log's offset and position.
//
// TODO: Implement recovery for ungraceful shutdowns. i.e. App crashes before truncating back file
type index struct {
	file *os.File
	mmap mmap.MMap
	size uint64
}

// newIndex creates a new log index based on the provided file and configuration.
func newIndex(f *os.File, c LogConfig) (*index, error) {
	idx := &index{
		file: f,
	}

	fileInfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	idx.size = uint64(fileInfo.Size())

	// Since we use memory mapping, we can't adjust the length of the file
	// later on without corruption. So we max out the length now and will handle
	// the truncating back to actual size during Close()
	if err = os.Truncate(
		f.Name(), int64(c.GetSegment().MaxIndexBytes),
	); err != nil {
		return nil, err
	}

	if idx.mmap, err = mmap.Map(
		idx.file,
		mmap.RDWR,
		0,
	); err != nil {
		return nil, err
	}
	return idx, nil
}

// Close closes the log index, ensuring proper flushing, syncing, and truncating of the file.
func (i *index) Close() error {
	// Making sure data is synced to disk
	if err := i.mmap.Flush(); err != nil {
		return err
	}
	if err := i.file.Sync(); err != nil {
		return err
	}
	// Truncating back the file to actual size to prevent corruption when Closing()
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return i.file.Close()
}

// Read retrieves the offset and position values from the log index at the specified position.
func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, io.EOF
	}

	// We use -1 as last position. Just like python
	if in == -1 {
		out = uint32((i.size / entryWidth) - 1)
	} else {
		out = uint32(in)
	}

	pos = uint64(out) * entryWidth
	if i.size < pos+entryWidth {
		return 0, 0, io.EOF
	}

	out = encoding.Uint32(i.mmap[pos : pos+offsetWidth])
	pos = encoding.Uint64(i.mmap[pos+offsetWidth : pos+entryWidth])
	return out, pos, nil
}

// Write appends a new entry to the log index with the given offset and position values.
func (i *index) Write(offset uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+entryWidth {
		return io.EOF
	}

	encoding.PutUint32(i.mmap[i.size:i.size+offsetWidth], offset)
	encoding.PutUint64(i.mmap[i.size+offsetWidth:i.size+entryWidth], pos)
	i.size += entryWidth
	return nil
}

// Name returns the name of the file associated with the log index.
func (i *index) Name() string {
	return i.file.Name()
}
