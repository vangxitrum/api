package s3_client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	PART_SIZE   = 100 * 1000 * 1000
	CONCURRENCY = 5
)

type S3Client struct {
	s3BucketName  string
	client        *s3.Client
	presignClient *s3.PresignClient
	logger        *slog.Logger
}

type S3ClientOption func(*S3Client)

func NewS3Client(
	key, secret, region, bucketName, endPoint string,
	opts ...S3ClientOption,
) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endPoint)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	presignClient := s3.NewPresignClient(client)

	helper := &S3Client{
		s3BucketName:  bucketName,
		client:        client,
		presignClient: presignClient,
		logger:        slog.Default(),
	}
	for _, opt := range opts {
		opt(helper)
	}

	buckets, err := client.ListBuckets(
		context.Background(),
		&s3.ListBucketsInput{},
	)
	if err != nil {
		return nil, err
	}

	for _, b := range buckets.Buckets {
		if *b.Name == bucketName {
			return helper, nil
		}
	}

	return nil, fmt.Errorf("bucket %s not found", bucketName)
}

func (bs *S3Client) Upload(
	ctx context.Context,
	key string,
	reader io.Reader,
) error {
	start := time.Now()
	defer func() {
		bs.logger.Debug(
			"s3 upload",
			slog.Any("runtime", time.Since(start).Seconds()),
			slog.Any("key", key),
		)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	newUploader := manager.NewUploader(bs.client, func(u *manager.Uploader) {
		u.PartSize = PART_SIZE
		u.Concurrency = CONCURRENCY
	})
	if _, err := newUploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bs.s3BucketName),
		Key:    aws.String(key),
		Body:   reader,
	}); err != nil {
		return err
	}

	return nil
}

func (bs *S3Client) Download(ctx context.Context, key string) (io.Reader, error) {
	start := time.Now()
	defer func() {
		bs.logger.Debug(
			"s3 download",
			slog.Any("runtime", time.Since(start).Seconds()),
			slog.Any("key", key),
		)
	}()

	resp, err := bs.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bs.s3BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (bs *S3Client) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		bs.logger.Debug(
			"s3 delete",
			slog.Any("runtime", time.Since(start).Seconds()),
			slog.Any("key", key),
		)
	}()

	_, err := bs.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bs.s3BucketName),
		Key:    aws.String(key),
	})

	return err
}

func (bs *S3Client) IsExist(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		bs.logger.Debug(
			"s3 is exist",
			slog.Any("runtime", time.Since(start).Seconds()),
			slog.Any("key", key),
		)
	}()

	_, err := bs.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(bs.s3BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (bs *S3Client) GetPresignURL(ctx context.Context, key string) (string, error) {
	start := time.Now()
	defer func() {
		bs.logger.Warn(
			"s3 presign url",
			slog.Any("runtime", time.Since(start).Seconds()),
			slog.Any("key", key),
		)
	}()

	req, err := bs.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bs.s3BucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(time.Second*3600))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func WithLogger(logger *slog.Logger) S3ClientOption {
	return func(bs *S3Client) {
		if logger != nil {
			bs.logger = logger
		}
	}
}
