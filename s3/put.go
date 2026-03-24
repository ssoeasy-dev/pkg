package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/ssoeasy-dev/pkg/errors"
)

type PutResult struct {
	ETag string
}

func (c *Client) Put(ctx context.Context, key string, r io.Reader, contentType *string) (*PutResult, error) {
	if r == nil {
		return nil, errors.New(errors.ErrInvalidArgument, "file reader is nil")
	}
	if key == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "key cannot be empty")
	}

	out, err := c.tmClient.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: contentType,
	})
	if err != nil {
		return nil, errors.New(errors.ErrCreationFailed, "failed to upload object %q: %v", key, err)
	}

	return &PutResult{ETag: aws.ToString(out.ETag)}, nil
}
