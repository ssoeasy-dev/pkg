package s3

import (
	"context"
	"testing"
	"time"

	"github.com/ssoeasy-dev/pkg/errors"
)

func TestClient_Presign_InvalidArgs(t *testing.T) {
	c := &Client{}
	ctx := context.Background()

	_, err := c.Presign(ctx, "", 1*time.Hour)
	if !errors.Is(err, errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}

	_, err = c.Presign(ctx, "key", 0)
	if !errors.Is(err, errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}
