package io

import (
	"context"
	"io"
)

func NewReaderContext(ctx context.Context, reader io.Reader) io.Reader {
	return &readerContext{
		ctx, reader,
	}
}

type readerContext struct {
	ctx    context.Context
	reader io.Reader
}

func (rc *readerContext) Read(buf []byte) (n int, err error) {
	if err := rc.ctx.Err(); err != nil {
		return 0, err
	}
	return rc.reader.Read(buf)
}
