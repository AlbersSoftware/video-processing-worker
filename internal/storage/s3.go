package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	client *s3.Client
}

func NewS3Storage(ctx context.Context, region string) (*S3Storage, error) {

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"load AWS configuration: %w",
			err,
		)
	}

	return &S3Storage{
		client: s3.NewFromConfig(cfg),
	}, nil
}

func (storage *S3Storage) DownloadObject(
	ctx context.Context,
	bucket string,
	key string,
	destination string,
) error {

	if err := os.MkdirAll(
		filepath.Dir(destination),
		0755,
	); err != nil {
		return fmt.Errorf(
			"create download directory: %w",
			err,
		)
	}

	result, err := storage.client.GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)

	if err != nil {
		return fmt.Errorf(
			"download s3://%s/%s: %w",
			bucket,
			key,
			err,
		)
	}

	defer result.Body.Close()

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf(
			"create destination file %s: %w",
			destination,
			err,
		)
	}

	defer file.Close()

	if _, err := io.Copy(
		file,
		result.Body,
	); err != nil {
		return fmt.Errorf(
			"write downloaded object to %s: %w",
			destination,
			err,
		)
	}

	return nil
}

func (storage *S3Storage) UploadFile(
	ctx context.Context,
	bucket string,
	key string,
	filePath string,
	contentType string,
) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf(
			"open upload file %s: %w",
			filePath,
			err,
		)
	}

	defer file.Close()

	_, err = storage.client.PutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        file,
			ContentType: aws.String(contentType),
		},
	)

	if err != nil {
		return fmt.Errorf(
			"upload %s to s3://%s/%s: %w",
			filePath,
			bucket,
			key,
			err,
		)
	}

	return nil
}
