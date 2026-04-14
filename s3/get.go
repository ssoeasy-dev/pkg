package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/ssoeasy-dev/pkg/errors"
)

// ObjectMetadata contains metadata about an S3 object.
type ObjectMetadata struct {
	ETag          string
	ContentType   string
	ContentLength int64
	Range         string // only present when a range was requested
}

// Get retrieves an object from the bucket.
// If rangeHeader is not nil, it must be a valid HTTP Range header (e.g., "bytes=0-1023").
// The caller is responsible for closing the returned ReadCloser.
func (c *Client) Get(ctx context.Context, key string, rangeHeader *string) (io.ReadCloser, *ObjectMetadata, error) {
	if key == "" {
		return nil, nil, errors.New(errors.ErrInvalidArgument, "key cannot be empty")
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	if rangeHeader != nil && *rangeHeader != "" {
		input.Range = rangeHeader
	}

	out, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return nil, nil, errors.NewWrapf(errors.ErrNotFound, err, "object %q not found", key)
		}
		return nil, nil, errors.NewWrapf(errors.ErrInternal, err, "failed to get object %q: %v", key, err)
	}

	meta := &ObjectMetadata{
		ETag:          aws.ToString(out.ETag),
		ContentType:   aws.ToString(out.ContentType),
		ContentLength: aws.ToInt64(out.ContentLength),
		Range:         aws.ToString(out.ContentRange),
	}
	return out.Body, meta, nil
}
