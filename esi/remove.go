package esi

import (
	"context"
	"regexp"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const remove = "remove"

var closeRemove = regexp.MustCompile("</esi:remove>")

type removeTag struct {
	*baseTag
}

func (r *removeTag) Process(ctx context.Context, b []byte, req *fsthttp.Request) ([]byte, int) {
	closeIdx := closeRemove.FindIndex(b)
	if closeIdx == nil {
		return []byte{}, len(b)
	}

	r.length = closeIdx[1]

	return []byte{}, r.length
}

func (*removeTag) HasClose(b []byte) bool {
	return closeRemove.FindIndex(b) != nil
}

func (*removeTag) GetClosePosition(b []byte) int {
	if idx := closeRemove.FindIndex(b); idx != nil {
		return idx[1]
	}

	return 0
}
