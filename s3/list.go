package s3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ssoeasy-dev/pkg/errors"
)

// ListResult represents a single object returned by List.
type ListResult struct {
	Key       string
	Size      int64
	ETag      string
	UpdatedAt time.Time
}

// List returns all objects with the given prefix, automatically handling pagination.
// Warning: for buckets with millions of objects, this method may consume a lot of memory.
// For such cases, consider using ListPages for iterative processing.
func (c *Client) List(ctx context.Context, prefix string) ([]ListResult, error) {
	var results []ListResult
	err := c.ListPages(ctx, prefix, func(page []ListResult) bool {
		results = append(results, page...)
		return true // continue to next page
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ListPages iterates over objects with the given prefix, calling the provided function
// for each page of results. The function should return true to continue, false to stop.
// This method is memory-efficient for large buckets.
func (c *Client) ListPages(ctx context.Context, prefix string, fn func([]ListResult) bool) error {
	var continuationToken *string
	for {
		out, err := c.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(c.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return errors.NewWrapf(errors.ErrInternal, err, "failed to list objects with prefix %q: %v", prefix, err)
		}

		page := make([]ListResult, 0, len(out.Contents))
		for _, obj := range out.Contents {
			page = append(page, ListResult{
				Key:       aws.ToString(obj.Key),
				Size:      aws.ToInt64(obj.Size),
				ETag:      aws.ToString(obj.ETag),
				UpdatedAt: aws.ToTime(obj.LastModified),
			})
		}
		if !fn(page) {
			break
		}

		if !aws.ToBool(out.IsTruncated) {
			break
		}
		continuationToken = out.NextContinuationToken
	}
	return nil
}
