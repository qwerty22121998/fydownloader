package core

import (
	"fmt"
	"io"
)

type FileHeader struct {
	ContentLength int
	AcceptRanges  string
	FileName      string
}

type FileChunk struct {
	Index int
	From  int
	To    int
}

func (p FileChunk) String() string {
	return fmt.Sprintf("bytes=%d-%d", p.From, p.To)
}

type ChunkProcessor struct {
	Chunk  FileChunk
	Index  int
	Reader io.ReadCloser
}

func (c *ChunkProcessor) Close() {
	c.Reader.Close()
}
