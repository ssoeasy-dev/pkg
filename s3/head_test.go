package s3

import (
	"context"
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
)

func TestClient_Head_InvalidKey(t *testing.T) {
	c := &Client{}
	ctx := context.Background()
	_, err := c.Head(ctx, "")
	if !errors.Is(err, errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}
