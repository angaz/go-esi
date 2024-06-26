package esi

import (
	"context"
	"regexp"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const comment = "comment"

var closeComment = regexp.MustCompile("/>((\n| +)+)?")

type commentTag struct {
	*baseTag
}

// Input (e.g. comment text="This is a comment." />).
func (c *commentTag) Process(ctx context.Context, b []byte, req *fsthttp.Request) ([]byte, int) {
	found := closeComment.FindIndex(b)
	if found == nil {
		return nil, len(b)
	}

	return []byte{}, found[1]
}

func (*commentTag) HasClose(b []byte) bool {
	return closeComment.FindIndex(b) != nil
}

func (*commentTag) GetClosePosition(b []byte) int {
	if idx := closeComment.FindIndex(b); idx != nil {
		return idx[1]
	}

	return 0
}
