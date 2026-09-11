package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"video-processing-worker/internal/model"
	"video-processing-worker/internal/storage"
)

func resolveSource(
	ctx context.Context,
	job model.VideoProcessingJob,
) (string, func(), error) {

	if job.Source != "" {
		return job.Source, func() {}, nil
	}

	workspace, err := os.MkdirTemp(
		"",
		"kinorify-video-*",
	)

	if err != nil {
		return "", func() {}, fmt.Errorf(
			"create temporary workspace: %w",
			err,
		)
	}

	cleanup := func() {

		if err := os.RemoveAll(
			workspace,
		); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"failed to remove temporary workspace %s: %v\n",
				workspace,
				err,
			)
		}
	}

	sourceFile := filepath.Join(
		workspace,
		"source",
		filepath.Base(
			job.SourceStorageKey,
		),
	)

	s3Storage, err := storage.NewS3Storage(
		ctx,
		"us-east-1",
	)

	if err != nil {
		cleanup()

		return "", func() {}, fmt.Errorf(
			"create S3 storage client: %w",
			err,
		)
	}

	fmt.Printf(
		"downloading source from s3://%s/%s\n",
		job.SourceBucket,
		job.SourceStorageKey,
	)

	if err := s3Storage.DownloadObject(
		ctx,
		job.SourceBucket,
		job.SourceStorageKey,
		sourceFile,
	); err != nil {
		cleanup()

		return "", func() {}, err
	}

	fmt.Printf(
		"downloaded source to %s\n",
		sourceFile,
	)

	return sourceFile, cleanup, nil
}
