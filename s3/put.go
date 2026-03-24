package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ssoeasy-dev/pkg/errors"
)

// PutResult contains the result of a Put operation.
type PutResult struct {
	ETag string
}

// Put uploads an object to the bucket with the given key.
// If contentType is nil, no Content-Type header is set.
// The method uses s3manager.Uploader internally to handle large files efficiently
// and retry failed parts automatically.
func (c *Client) Put(ctx context.Context, key string, r io.Reader, contentType *string) (*PutResult, error) {
	if r == nil {
		return nil, errors.New(errors.ErrInvalidArgument, "file reader is nil")
	}
	if key == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "key cannot be empty")
	}

	uploader := manager.NewUploader(c.s3Client)
	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   r,
	}
	if contentType != nil {
		input.ContentType = contentType
	}

	out, err := uploader.Upload(ctx, input)
	if err != nil {
		return nil, errors.New(errors.ErrCreationFailed, "failed to upload object %q: %v", key, err)
	}

	return &PutResult{ETag: *out.ETag}, nil
}
