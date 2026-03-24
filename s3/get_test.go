package s3

import (
	"context"
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
)

func TestClient_Get_InvalidKey(t *testing.T) {
	c := &Client{}
	ctx := context.Background()
	_, _, err := c.Get(ctx, "", nil)
	if !errors.Is(err, errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}
