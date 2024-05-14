package esi

import (
	"context"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

type (
	Tag interface {
		Process(context.Context, []byte, *fsthttp.Request) ([]byte, int)
		HasClose([]byte) bool
		GetClosePosition([]byte) int
	}

	baseTag struct {
		length int
	}
)

func newBaseTag() *baseTag {
	return &baseTag{length: 0}
}

func (b *baseTag) Process(_ context.Context, content []byte, _ *fsthttp.Request) ([]byte, int) {
	return []byte{}, len(content)
}
