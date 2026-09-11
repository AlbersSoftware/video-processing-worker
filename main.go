package main

import (
	"context"
	"fmt"
	"os"

	"video-processing-worker/internal/model"
)

func main() {

	ctx := context.Background()

	job, err := loadJob(
		os.Args,
	)

	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"failed to load video processing job: %v\n",
			err,
		)

		os.Exit(1)
	}

	result, err := runJob(
		ctx,
		job,
	)

	if err != nil {

		fmt.Fprintf(
			os.Stderr,
			"video processing job failed: %v\n",
			err,
		)

		failureMessage := err.Error()

		failureResult := model.VideoProcessingResult{
			Successful:     false,
			FailureMessage: &failureMessage,
			Renditions:     []model.VideoRenditionResult{},
			Manifests:      []model.VideoManifestResult{},
		}

		if reportErr := reportProcessingResult(
			ctx,
			job,
			failureResult,
		); reportErr != nil {

			fmt.Fprintf(
				os.Stderr,
				"failed to report video processing failure: %v\n",
				reportErr,
			)
		}

		os.Exit(1)
	}

	if err := reportProcessingResult(
		ctx,
		job,
		result,
	); err != nil {

		fmt.Fprintf(
			os.Stderr,
			"failed to report successful video processing result: %v\n",
			err,
		)

		os.Exit(1)
	}

	fmt.Printf(
		"video processing job %s completed successfully for media %s\n",
		job.ProcessingJobID,
		job.MediaID,
	)
}
