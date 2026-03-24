package s3

import (
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
)

func TestNewClient_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{"nil config", nil},
		{"empty bucket", &Config{Bucket: ""}},
		{"empty access key", &Config{Bucket: "test", AccessKey: "", SecretKey: "secret"}},
		{"empty secret key", &Config{Bucket: "test", AccessKey: "key", SecretKey: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.cfg)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, errors.ErrInvalidArgument) {
				t.Errorf("expected ErrInvalidArgument, got %v", err)
			}
		})
	}
}

func TestNewClient_ValidConfig(t *testing.T) {
	cfg := &Config{
		Endpoint:       "http://example.com",
		Region:         "us-east-1",
		AccessKey:      "valid-key",
		SecretKey:      "valid-secret",
		Bucket:         "my-bucket",
		ForcePathStyle: true,
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.bucket != "my-bucket" {
		t.Errorf("bucket = %q, want %q", client.bucket, "my-bucket")
	}
	if client.s3Client == nil {
		t.Error("s3Client is nil")
	}
	if client.presignClient == nil {
		t.Error("presignClient is nil")
	}
}
