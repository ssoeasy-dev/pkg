package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ssoeasy-dev/pkg/errors"
)

// Client is a generic S3 client that works with any S3-compatible storage.
type Client struct {
	s3Client      *s3.Client
	bucket        string
	presignClient *s3.PresignClient
	tmClient      *transfermanager.Client // новый менеджер загрузок
}

// NewClient creates a new S3 client with the given configuration.
func NewClient(cfg *Config) (*Client, error) {
	return NewClientWithContext(context.Background(), cfg)
}

// NewClientWithContext creates a new S3 client with the given configuration and context.
func NewClientWithContext(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New(errors.ErrInvalidArgument, "config is nil")
	}
	if cfg.Bucket == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "bucket name is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "access key and secret key are required")
	}

	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.ForcePathStyle {
			o.UsePathStyle = true
		}
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	// Создаём клиент transfer manager с настройками по умолчанию.
	// При необходимости можно настроить concurrency и part size.
	tmClient := transfermanager.New(client)

	return &Client{
		s3Client:      client,
		bucket:        cfg.Bucket,
		presignClient: s3.NewPresignClient(client),
		tmClient:      tmClient,
	}, nil
}
