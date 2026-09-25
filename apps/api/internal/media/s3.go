package media

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage stores objects in any S3-compatible service (AWS S3, Cloudflare R2,
// MinIO, IDCloudHost, …). The bucket stays private: files are streamed through
// the API so the published/unpublished access rule is enforced in one place.
type S3Storage struct {
	client *minio.Client
	bucket string
}

type S3Config struct {
	Endpoint  string // host[:port], no scheme
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func NewS3(ctx context.Context, c S3Config) (*S3Storage, error) {
	if c.Endpoint == "" || c.Bucket == "" || c.AccessKey == "" || c.SecretKey == "" {
		return nil, errors.New("S3 storage needs S3_ENDPOINT, S3_BUCKET, STORAGE_ACCESS_KEY and STORAGE_SECRET_KEY")
	}
	cl, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: c.UseSSL,
		Region: c.Region,
	})
	if err != nil {
		return nil, err
	}
	ok, err := cl.BucketExists(ctx, c.Bucket)
	if err != nil {
		return nil, fmt.Errorf("S3 bucket check: %w", err)
	}
	if !ok {
		if err := cl.MakeBucket(ctx, c.Bucket, minio.MakeBucketOptions{Region: c.Region}); err != nil {
			return nil, fmt.Errorf("S3 create bucket: %w", err)
		}
	}
	return &S3Storage{client: cl, bucket: c.Bucket}, nil
}

func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, r, -1, minio.PutObjectOptions{})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (s *S3Storage) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, err
	}
	return obj, nil
}
