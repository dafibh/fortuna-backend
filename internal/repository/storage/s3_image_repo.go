package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	cfg "github.com/dafibh/fortuna/fortuna-backend/internal/config"
)

// S3ImageRepository implements ImageRepository using AWS S3
type S3ImageRepository struct {
	client       *s3.Client
	presigner    *s3.PresignClient
	bucket       string
}

// NewS3ImageRepository creates a new S3 image repository
func NewS3ImageRepository(ctx context.Context, s3cfg cfg.S3Config) (*S3ImageRepository, error) {
	// Build AWS config options
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(s3cfg.Region),
	}

	// Add credentials if provided
	if s3cfg.AccessKeyID != "" && s3cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s3cfg.AccessKeyID,
				s3cfg.SecretAccessKey,
				"",
			),
		))
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with optional endpoint override for MinIO/LocalStack
	var client *s3.Client
	if s3cfg.Endpoint != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s3cfg.Endpoint)
			o.UsePathStyle = true // Required for MinIO
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	repo := &S3ImageRepository{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    s3cfg.Bucket,
	}

	// Verify connectivity by checking if bucket exists (don't create public policy)
	if err := repo.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// ensureBucket creates the bucket if it doesn't exist (NO public policy - private bucket)
func (r *S3ImageRepository) ensureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(r.bucket),
	})
	if err == nil {
		return nil // Bucket exists and we have access
	}

	// Check if it's a "not found" error vs permission/other error
	var notFound *types.NotFound
	if !errors.As(err, &notFound) {
		// Not a "not found" error - could be permission denied or other issue
		// For S3, we also check for the specific status code pattern
		var noSuchBucket *types.NoSuchBucket
		if !errors.As(err, &noSuchBucket) {
			// This is likely a permission error or connectivity issue, not "bucket doesn't exist"
			return fmt.Errorf("failed to check bucket (may be permission denied): %w", err)
		}
	}

	// Bucket doesn't exist - try to create it (NO public policy - remains private)
	_, err = r.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(r.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// Upload uploads data to S3 storage and returns the object path (not URL)
func (r *S3ImageRepository) Upload(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error) {
	// If size is unknown, read all data into memory
	var body io.Reader = data
	if size < 0 {
		buf, err := io.ReadAll(data)
		if err != nil {
			return "", fmt.Errorf("failed to read data: %w", err)
		}
		size = int64(len(buf))
		body = bytes.NewReader(buf)
	}

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(objectPath),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	// Return object path (not URL) - presigned URLs generated on demand
	return objectPath, nil
}

// Delete removes an object from S3 storage
func (r *S3ImageRepository) Delete(ctx context.Context, objectPath string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(objectPath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GeneratePresignedURL generates a presigned GET URL for temporary access
func (r *S3ImageRepository) GeneratePresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	presignedReq, err := r.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(objectPath),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedReq.URL, nil
}
