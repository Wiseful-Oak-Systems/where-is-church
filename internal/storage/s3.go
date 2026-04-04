package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Store implements Store using AWS S3 or S3-compatible services (MinIO, DigitalOcean Spaces, etc.)
type S3Store struct {
	client   *s3.Client
	bucket   string
	endpoint string
}

func NewS3Store(bucket, region, endpoint, accessKey, secretKey string) (*S3Store, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is required when STORAGE_DRIVER=s3")
	}

	ctx := context.Background()
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	if accessKey != "" && secretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var s3Opts []func(*s3.Options)
	if endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // needed for MinIO and most S3-compatible services
		})
	}

	client := s3.NewFromConfig(cfg, s3Opts...)

	return &S3Store{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
	}, nil
}

func (s *S3Store) Save(ctx context.Context, key string, reader io.Reader) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}
	return key, nil
}

func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get from S3: %w", err)
	}
	return output.Body, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

func (s *S3Store) URL(key string) string {
	if s.endpoint != "" {
		endpoint := strings.TrimRight(s.endpoint, "/")
		return fmt.Sprintf("%s/%s/%s", endpoint, s.bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
}
