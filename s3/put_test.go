package s3

import (
	"context"
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
)

func TestClient_Put_InvalidArgs(t *testing.T) {
	c := &Client{}
	ctx := context.Background()

	_, err := c.Put(ctx, "", nil, nil)
	if !errors.Is(err, errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}

	_, err = c.Put(ctx, "key", nil, nil)
	if !errors.Is(err, errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}
