package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client — generic S3 клиент поверх aws-sdk-go-v2.
// Работает с любым S3-совместимым хранилищем (AWS, Tinkoff Cloud, MinIO и т.д.).
type Client struct {
	S3     *s3.Client
	bucket string
}

// NewClient создаёт и возвращает готовый S3 Client.
func NewClient(cfg *Config) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.ForcePathStyle {
			o.UsePathStyle = true
		}
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationUnset
	})

	return &Client{
		S3:     client,
		bucket: cfg.Bucket,
	}, nil
}

type Object struct {
	Key           string
	ContentType   *string
	ContentLength *int64
	Range         *string
	ETag          *string
}

// Put загружает объект по ключу key с указанным contentType.
func (c *Client) Put(ctx context.Context, file io.Reader, obj *Object) error {
	if file == nil {
		return fmt.Errorf("Error")
	}
	if obj == nil {
		return fmt.Errorf("Error")
	}

	o, err := c.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(obj.Key),
		Body:        file,
		ContentType: obj.ContentType,
	})

	obj.ETag = o.ETag

	if err != nil {
		return fmt.Errorf("s3: put %q: %w", obj.Key, err)
	}
	return nil
}

type ListResult struct {
	Key       string
	Size      int64
	ETag      string
	UpdatedAt time.Time
}

// List возвращает объекты по ключу key.
func (c *Client) List(ctx context.Context, key string) ([]ListResult, error) {
	var continuationToken *string
	var res []ListResult
	for {
		listOut, err := c.S3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(c.bucket),
			Prefix:            aws.String(key),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, err
		}

		for _, obj := range listOut.Contents {
			res = append(res, ListResult{
				Key:       aws.ToString(obj.Key),
				Size:      aws.ToInt64(obj.Size),
				ETag:      aws.ToString(obj.ETag),
				UpdatedAt: aws.ToTime(obj.LastModified),
			})
		}

		if !aws.ToBool(listOut.IsTruncated) {
			break
		}
		continuationToken = listOut.NextContinuationToken
	}

	return res, nil
}

// Get возвращает объект по ключу key.
// rangeHeader — значение HTTP-заголовка Range (например "bytes=0-1023").
// Передайте пустую строку для получения полного объекта.
// Вызывающий обязан закрыть Body у возвращённого объекта.
func (c *Client) Get(ctx context.Context, obj *Object) (*io.ReadCloser, error) {
	out, err := c.S3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(obj.Key),
		Range:  obj.Range,
	})
	if err != nil {
		return nil, fmt.Errorf("s3: get %q: %w", obj.Key, err)
	}

	obj.ETag = out.ETag
	obj.Range = out.ContentRange
	obj.ContentType = out.ContentType
	obj.ContentLength = out.ContentLength

	return &out.Body, nil
}

// Head возвращает метаданные объекта без скачивания тела.
// Удобен для получения Content-Length, Content-Type, ETag.
func (c *Client) Head(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	out, err := c.S3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: head %q: %w", key, err)
	}
	return out, nil
}
