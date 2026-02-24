package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Backend stores files in an S3-compatible object store.
type S3Backend struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
	publicURL  string
	usePresign bool
	presignTTL time.Duration
}

// S3Config holds the configuration for an S3-compatible storage backend.
type S3Config struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey  string
	PublicURL       string
	UsePresign      bool
}

func NewS3Backend(ctx context.Context, cfg S3Config) (*S3Backend, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	return &S3Backend{
		client:     client,
		presigner:  s3.NewPresignClient(client),
		bucket:     cfg.Bucket,
		publicURL:  strings.TrimSuffix(cfg.PublicURL, "/"),
		usePresign: cfg.UsePresign,
		presignTTL: 1 * time.Hour,
	}, nil
}

func (b *S3Backend) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(b.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	}
	if _, err := b.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("uploading to S3: %w", err)
	}
	return nil
}

func (b *S3Backend) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	output, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("getting from S3: %w", err)
	}
	return output.Body, aws.ToInt64(output.ContentLength), nil
}

func (b *S3Backend) Delete(ctx context.Context, key string) error {
	if _, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("deleting from S3: %w", err)
	}
	return nil
}

func (b *S3Backend) URL(key string) string {
	if b.usePresign {
		// For presigned: still use the API endpoint for redirect
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			return ""
		}
		filename := parts[len(parts)-1]
		id := strings.TrimSuffix(filename, filepath.Ext(filename))
		return "/api/v1/images/" + id
	}
	// Public bucket URL — direct access
	return b.publicURL + "/" + key
}

func (b *S3Backend) PresignedURL(ctx context.Context, key string) (string, error) {
	output, err := b.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(b.presignTTL))
	if err != nil {
		return "", fmt.Errorf("presigning S3 URL: %w", err)
	}
	return output.URL, nil
}

func (b *S3Backend) Name() string { return "s3" }
