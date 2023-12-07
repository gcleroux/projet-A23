package distributedLog

import (
	"encoding/binary"
	"fmt"

	"github.com/gcleroux/projet-A23/src/log"
	"github.com/hashicorp/raft"
)

var encoding = binary.BigEndian

const (
	lenWidth   uint64 = 8
	entryWidth uint64 = 12
)

type Config struct {
	Raft struct {
		raft.Config
		StreamLayer *StreamLayer
		Bootstrap   bool
	}
	log.Config
}

func (c *Config) Init() {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = 1024
	}
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = 1024
	}
	if c.Segment.InitialOffset == 0 {
		c.Segment.InitialOffset = 1
	}
}

func (c *Config) Validate() error {
	if c.Segment.MaxStoreBytes < lenWidth {
		return fmt.Errorf("MaxStoreBytes=%d, can't be < %d", c.Segment.MaxStoreBytes, lenWidth)
	}
	if c.Segment.MaxIndexBytes < entryWidth {
		return fmt.Errorf("MaxIndexBytes=%d, can't be < %d", c.Segment.MaxIndexBytes, entryWidth)
	}
	if c.Segment.InitialOffset == 0 {
		return fmt.Errorf("InitialOffset=%d, offset must start at 1 to be compliant with raft API", c.Segment.InitialOffset)
	}
	return nil
}

func (c *Config) GetSegment() *log.SegmentConfig {
	return &c.Segment
}
