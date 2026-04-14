package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/ssoeasy-dev/pkg/errors"
)

// Head returns metadata about an object without downloading its body.
func (c *Client) Head(ctx context.Context, key string) (*ObjectMetadata, error) {
	if key == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "key cannot be empty")
	}

	out, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return nil, errors.NewWrapf(errors.ErrNotFound, err, "object %q not found", key)
		}
		return nil, errors.NewWrapf(errors.ErrInternal, err, "failed to head object %q: %v", key, err)
	}

	return &ObjectMetadata{
		ETag:          aws.ToString(out.ETag),
		ContentType:   aws.ToString(out.ContentType),
		ContentLength: aws.ToInt64(out.ContentLength),
		// Range not applicable for Head
	}, nil
}
