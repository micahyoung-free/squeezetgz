package gzip

import (
	"compress/gzip"
	"io"
)

const (
	BestCompression = gzip.BestCompression
)

type Writer = gzip.Writer

type Reader = gzip.Reader

func NewWriterLevel(w io.Writer, level int) (*Writer, error) {
	return gzip.NewWriterLevel(w, level)
}

func NewReader(r io.Reader) (*Reader, error) {
	return gzip.NewReader(r)
}
