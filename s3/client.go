package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client — generic S3 клиент поверх aws-sdk-go-v2.
// Работает с любым S3-совместимым хранилищем (AWS, Tinkoff Cloud, MinIO и т.д.).
type Client struct {
	s3     *s3.Client
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
	})

	return &Client{
		s3:     client,
		bucket: cfg.Bucket,
	}, nil
}

// Put загружает объект по ключу key с указанным contentType.
func (c *Client) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3: put %q: %w", key, err)
	}
	return nil
}

// Get возвращает объект по ключу key.
// rangeHeader — значение HTTP-заголовка Range (например "bytes=0-1023").
// Передайте пустую строку для получения полного объекта.
// Вызывающий обязан закрыть Body у возвращённого объекта.
func (c *Client) Get(ctx context.Context, key string, rangeHeader string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	if rangeHeader != "" {
		input.Range = aws.String(rangeHeader)
	}

	out, err := c.s3.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("s3: get %q: %w", key, err)
	}
	return out, nil
}

// Head возвращает метаданные объекта без скачивания тела.
// Удобен для получения Content-Length, Content-Type, ETag.
func (c *Client) Head(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	out, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: head %q: %w", key, err)
	}
	return out, nil
}
