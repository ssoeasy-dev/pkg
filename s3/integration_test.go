//go:build integration

package s3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ssoeasy-dev/pkg/errors"
)

func setupClient(t *testing.T) (*Client, func()) {
	t.Helper()
	ctx := context.Background()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	minioContainer, err := minio.Run(ctx,
		"minio/minio:latest",
		minio.WithUsername("minioadmin"),
		minio.WithPassword("minioadmin"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("9000/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	endpoint, err := minioContainer.Endpoint(ctx, "http")
	require.NoError(t, err)

	cfg := &Config{
		Endpoint:       endpoint,
		Region:         "us-east-1", // обязательно для AWS SDK v2
		AccessKey:      "minioadmin",
		SecretKey:      "minioadmin",
		Bucket:         "test-bucket",
		ForcePathStyle: true,
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)

	_, err = client.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})
	require.NoError(t, err)

	cleanup := func() {
		cleanupBucket(t, client)
		_ = minioContainer.Terminate(ctx)
	}
	return client, cleanup
}

func cleanupBucket(t *testing.T, client *Client) {
	t.Helper()
	ctx := context.Background()
	objects, err := client.List(ctx, "")
	if err != nil {
		t.Logf("cleanup list error: %v", err)
		return
	}
	for _, obj := range objects {
		_, err := client.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(client.bucket),
			Key:    aws.String(obj.Key),
		})
		if err != nil {
			t.Logf("cleanup delete error for %s: %v", obj.Key, err)
		}
	}
}

func randomKey() string {
	return uuid.New().String()
}

func testData(size int) []byte {
	return bytes.Repeat([]byte("x"), size)
}

func TestPut(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("upload small object", func(t *testing.T) {
		key := randomKey()
		data := []byte("hello world")
		r := bytes.NewReader(data)

		res, err := client.Put(ctx, key, r, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, res.ETag)

		body, meta, err := client.Get(ctx, key, nil)
		require.NoError(t, err)
		defer body.Close()
		got, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, data, got)
		assert.Equal(t, int64(len(data)), meta.ContentLength)
	})

	t.Run("upload with contentType", func(t *testing.T) {
		key := randomKey()
		data := []byte(`{"foo":"bar"}`)
		contentType := "application/json"
		_, err := client.Put(ctx, key, bytes.NewReader(data), &contentType)
		require.NoError(t, err)

		meta, err := client.Head(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, contentType, meta.ContentType)
	})

	t.Run("upload large object", func(t *testing.T) {
		key := randomKey()
		data := testData(10 * 1024 * 1024)
		_, err := client.Put(ctx, key, bytes.NewReader(data), nil)
		require.NoError(t, err)

		meta, err := client.Head(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), meta.ContentLength)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		_, err := client.Put(ctx, "", nil, nil)
		assert.True(t, errors.Is(err, errors.ErrInvalidArgument))

		_, err = client.Put(ctx, "key", nil, nil)
		assert.True(t, errors.Is(err, errors.ErrInvalidArgument))
	})
}

func TestGet(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	ctx := context.Background()

	key := randomKey()
	data := []byte("abcdefghijklmnopqrstuvwxyz")
	_, err := client.Put(ctx, key, bytes.NewReader(data), nil)
	require.NoError(t, err)

	t.Run("get full object", func(t *testing.T) {
		body, meta, err := client.Get(ctx, key, nil)
		require.NoError(t, err)
		defer body.Close()

		got, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, data, got)
		assert.Equal(t, int64(len(data)), meta.ContentLength)
		assert.NotEmpty(t, meta.ETag)
	})

	t.Run("get with range", func(t *testing.T) {
		rangeHeader := "bytes=2-5"
		body, meta, err := client.Get(ctx, key, &rangeHeader)
		require.NoError(t, err)
		defer body.Close()

		got, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, []byte("cdef"), got)
		assert.Equal(t, "bytes 2-5/26", meta.Range)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		_, _, err := client.Get(ctx, "nonexistent", nil)
		assert.True(t, errors.Is(err, errors.ErrNotFound))
	})

	t.Run("get with empty key", func(t *testing.T) {
		_, _, err := client.Get(ctx, "", nil)
		assert.True(t, errors.Is(err, errors.ErrInvalidArgument))
	})
}

func TestHead(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	ctx := context.Background()

	key := randomKey()
	contentType := "text/plain"
	data := []byte("hello")
	_, err := client.Put(ctx, key, bytes.NewReader(data), &contentType)
	require.NoError(t, err)

	t.Run("head existing object", func(t *testing.T) {
		meta, err := client.Head(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), meta.ContentLength)
		assert.Equal(t, contentType, meta.ContentType)
		assert.NotEmpty(t, meta.ETag)
	})

	t.Run("head non-existent object", func(t *testing.T) {
		_, err := client.Head(ctx, "nonexistent")
		assert.True(t, errors.Is(err, errors.ErrNotFound))
	})

	t.Run("head with empty key", func(t *testing.T) {
		_, err := client.Head(ctx, "")
		assert.True(t, errors.Is(err, errors.ErrInvalidArgument))
	})
}

func TestList(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	ctx := context.Background()

	keys := []string{
		"a/1.txt", "a/2.txt", "a/3.txt",
		"b/1.txt", "b/2.txt",
		"c/1.txt",
		"root.txt",
	}
	for _, key := range keys {
		_, err := client.Put(ctx, key, strings.NewReader("data"), nil)
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		results, err := client.List(ctx, "")
		require.NoError(t, err)
		assert.Len(t, results, 7)
	})

	t.Run("list with prefix a/", func(t *testing.T) {
		results, err := client.List(ctx, "a/")
		require.NoError(t, err)
		assert.Len(t, results, 3)
		for _, r := range results {
			assert.True(t, strings.HasPrefix(r.Key, "a/"))
		}
	})

	t.Run("list with prefix that has no objects", func(t *testing.T) {
		results, err := client.List(ctx, "nonexistent/")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("list pagination", func(t *testing.T) {
		var allKeys []string
		err := client.ListPages(ctx, "", func(page []ListResult) bool {
			for _, obj := range page {
				allKeys = append(allKeys, obj.Key)
			}
			return true
		})
		require.NoError(t, err)
		assert.Len(t, allKeys, 7)
	})
}

func TestPresign(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	ctx := context.Background()

	key := randomKey()
	data := []byte("secret data")
	_, err := client.Put(ctx, key, bytes.NewReader(data), nil)
	require.NoError(t, err)

	t.Run("presign get", func(t *testing.T) {
		url, err := client.Presign(ctx, key, 5*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, url)

		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, data, body)
	})

	t.Run("presign with zero ttl", func(t *testing.T) {
		_, err := client.Presign(ctx, key, 0)
		assert.True(t, errors.Is(err, errors.ErrInvalidArgument))
	})

	t.Run("presign with empty key", func(t *testing.T) {
		_, err := client.Presign(ctx, "", 1*time.Hour)
		assert.True(t, errors.Is(err, errors.ErrInvalidArgument))
	})

	t.Run("presign non-existent key", func(t *testing.T) {
		_, err := client.Presign(ctx, "nonexistent", 1*time.Hour)
		require.NoError(t, err)
	})
}
