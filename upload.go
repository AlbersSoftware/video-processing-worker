package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"video-processing-worker/internal/model"
	"video-processing-worker/internal/storage"
)

func uploadPackage(
	ctx context.Context,
	job model.VideoProcessingJob,
	outputRoot string,
) error {

	packageRoot := filepath.Join(
		outputRoot,
		"package",
	)

	s3Storage, err := storage.NewS3Storage(
		ctx,
		"us-east-1",
	)

	if err != nil {
		return fmt.Errorf(
			"create S3 storage client: %w",
			err,
		)
	}

	fmt.Printf(
		"\nuploading packaged output to s3://%s/%s\n",
		job.DerivedBucket,
		job.DerivedStoragePrefix,
	)

	err = filepath.WalkDir(
		packageRoot,
		func(
			filePath string,
			entry os.DirEntry,
			walkErr error,
		) error {

			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(
				packageRoot,
				filePath,
			)

			if err != nil {
				return fmt.Errorf(
					"resolve relative package path for %s: %w",
					filePath,
					err,
				)
			}

			s3Key := buildDerivedStorageKey(
				job.DerivedStoragePrefix,
				relativePath,
			)

			contentType := contentTypeForFile(
				filePath,
			)

			fmt.Printf(
				"uploading %s -> s3://%s/%s [%s]\n",
				filePath,
				job.DerivedBucket,
				s3Key,
				contentType,
			)

			if err := s3Storage.UploadFile(
				ctx,
				job.DerivedBucket,
				s3Key,
				filePath,
				contentType,
			); err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return fmt.Errorf(
			"walk packaged output directory: %w",
			err,
		)
	}

	fmt.Println(
		"package upload completed successfully",
	)

	return nil
}

func buildDerivedStorageKey(
	derivedStoragePrefix string,
	relativePath string,
) string {

	return path.Join(
		strings.TrimSuffix(
			derivedStoragePrefix,
			"/",
		),
		"package",
		filepath.ToSlash(
			relativePath,
		),
	)
}

func contentTypeForFile(
	filePath string,
) string {

	switch strings.ToLower(
		filepath.Ext(filePath),
	) {

	case ".m3u8":
		return "application/vnd.apple.mpegurl"

	case ".mpd":
		return "application/dash+xml"

	case ".mp4":
		return "application/mp4"

	case ".m4s":
		return "application/octet-stream"

	default:
		return "application/octet-stream"
	}
}
