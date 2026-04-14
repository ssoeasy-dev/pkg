package s3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ssoeasy-dev/pkg/errors"
)

// Presign generates a pre-signed URL for the given object, valid for the specified duration.
func (c *Client) Presign(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if key == "" {
		return "", errors.New(errors.ErrInvalidArgument, "key cannot be empty")
	}
	if ttl <= 0 {
		return "", errors.New(errors.ErrInvalidArgument, "TTL must be positive")
	}

	out, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", errors.NewWrapf(errors.ErrInternal, err, "failed to presign %q: %v", key, err)
	}
	return out.URL, nil
}
