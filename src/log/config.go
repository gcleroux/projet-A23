package log

import "fmt"

type LogConfig interface {
	Init()
	Validate() error
	GetSegment() *SegmentConfig
}

type Config struct {
	Segment SegmentConfig
}

type SegmentConfig struct {
	MaxStoreBytes uint64
	MaxIndexBytes uint64
	InitialOffset uint64
}

func (c *Config) Init() {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = 1024
	}
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = 1024
	}
}

func (c *Config) Validate() error {
	if c.Segment.MaxStoreBytes < lenWidth {
		return fmt.Errorf("MaxStoreBytes=%d, can't be < %d", c.Segment.MaxStoreBytes, lenWidth)
	}
	if c.Segment.MaxIndexBytes < entryWidth {
		return fmt.Errorf("MaxIndexBytes=%d, can't be < %d", c.Segment.MaxIndexBytes, entryWidth)
	}
	return nil
}

func (c *Config) GetSegment() *SegmentConfig {
	return &c.Segment
}
