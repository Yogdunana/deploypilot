package backup

import (
	"context"
	"io"
	"time"
)

// StorageType defines the type of cloud storage backend.
type StorageType string

const (
	StorageLocal StorageType = "local"
	StorageS3    StorageType = "s3"
	StorageOSS   StorageType = "oss"
	StorageCOS   StorageType = "cos"
	StorageMinIO StorageType = "minio"
)

// StorageConfig holds cloud storage connection configuration.
type StorageConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Type        string `mapstructure:"type"`          // s3, oss, cos, minio
	Endpoint    string `mapstructure:"endpoint"`      // e.g. "https://s3.amazonaws.com"
	Region      string `mapstructure:"region"`        // e.g. "us-east-1"
	Bucket      string `mapstructure:"bucket"`        // bucket name
	Prefix      string `mapstructure:"prefix"`        // key prefix, e.g. "deploypilot/backups/"
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	UseSSL      bool   `mapstructure:"use_ssl"`       // default: true
	Encrypt     bool   `mapstructure:"encrypt"`       // enable AES-256-GCM encryption before upload
}

// StorageProvider defines the interface for cloud storage backends.
// Implementations support S3-compatible object storage (AWS S3, Aliyun OSS, Tencent COS, MinIO).
type StorageProvider interface {
	// Upload uploads a file to cloud storage.
	Upload(ctx context.Context, key string, reader io.Reader, size int64) error

	// Download downloads a file from cloud storage.
	Download(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// Delete removes a file from cloud storage.
	Delete(ctx context.Context, key string) error

	// List lists objects with the given prefix.
	List(ctx context.Context, prefix string) ([]StorageObject, error)

	// Exists checks if an object exists.
	Exists(ctx context.Context, key string) (bool, error)

	// Type returns the storage type.
	Type() StorageType
}

// StorageObject represents a cloud storage object.
type StorageObject struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag,omitempty"`
}
