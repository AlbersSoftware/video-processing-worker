package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"video-processing-worker/internal/model"
)

const mediaSvcBaseURLEnv = "MEDIA_SVC_BASE_URL"

const processingResultRequestTimeout = 30 * time.Second

const processingResultMaxAttempts = 3

const processingResultRetryDelay = 2 * time.Second

func reportProcessingResult(
	ctx context.Context,
	job model.VideoProcessingJob,
	result model.VideoProcessingResult,
) error {

	baseURL := strings.TrimSuffix(
		os.Getenv(mediaSvcBaseURLEnv),
		"/",
	)

	if baseURL == "" {
		return fmt.Errorf(
			"%s is required",
			mediaSvcBaseURLEnv,
		)
	}

	callbackURL := fmt.Sprintf(
		"%s/api/media/collection-media/videos/%s/processing-jobs/%s/result",
		baseURL,
		job.MediaID,
		job.ProcessingJobID,
	)

	requestBody, err := json.Marshal(
		result,
	)

	if err != nil {
		return fmt.Errorf(
			"serialize video processing result: %w",
			err,
		)
	}

	client := &http.Client{
		Timeout: processingResultRequestTimeout,
	}

	var lastErr error

	for attempt := 1; attempt <= processingResultMaxAttempts; attempt++ {

		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			callbackURL,
			bytes.NewReader(requestBody),
		)

		if err != nil {
			return fmt.Errorf(
				"create video processing result request: %w",
				err,
			)
		}

		request.Header.Set(
			"Content-Type",
			"application/json",
		)

		response, err := client.Do(
			request,
		)

		if err != nil {

			lastErr = fmt.Errorf(
				"send video processing result: %w",
				err,
			)

			fmt.Printf(
				"video processing callback attempt %d/%d failed for media %s processing job %s: %v\n",
				attempt,
				processingResultMaxAttempts,
				job.MediaID,
				job.ProcessingJobID,
				err,
			)

		} else {

			responseBody, readErr := io.ReadAll(
				response.Body,
			)

			response.Body.Close()

			if readErr != nil {

				lastErr = fmt.Errorf(
					"read video processing result response: %w",
					readErr,
				)

				fmt.Printf(
					"video processing callback attempt %d/%d failed while reading response for media %s processing job %s: %v\n",
					attempt,
					processingResultMaxAttempts,
					job.MediaID,
					job.ProcessingJobID,
					readErr,
				)

			} else if response.StatusCode >= http.StatusOK &&
				response.StatusCode < http.StatusMultipleChoices {

				if result.Successful &&
					len(result.Renditions) == 0 &&
					len(result.Manifests) == 0 {

					fmt.Printf(
						"video processing result reported successfully for media %s processing job %s: no renditions needed\n",
						job.MediaID,
						job.ProcessingJobID,
					)

					return nil
				}

				fmt.Printf(
					"video processing result reported successfully for media %s processing job %s\n",
					job.MediaID,
					job.ProcessingJobID,
				)

				return nil

			} else if response.StatusCode >= http.StatusInternalServerError {

				lastErr = fmt.Errorf(
					"media-svc rejected video processing result: status=%d body=%s",
					response.StatusCode,
					string(responseBody),
				)

				fmt.Printf(
					"video processing callback attempt %d/%d received server error for media %s processing job %s: status=%d\n",
					attempt,
					processingResultMaxAttempts,
					job.MediaID,
					job.ProcessingJobID,
					response.StatusCode,
				)

			} else {

				return fmt.Errorf(
					"media-svc rejected video processing result: status=%d body=%s",
					response.StatusCode,
					string(responseBody),
				)
			}
		}

		if attempt < processingResultMaxAttempts {

			select {
			case <-ctx.Done():
				return fmt.Errorf(
					"video processing result callback canceled: %w",
					ctx.Err(),
				)

			case <-time.After(processingResultRetryDelay):
			}
		}
	}

	return fmt.Errorf(
		"video processing result callback failed after %d attempts: %w",
		processingResultMaxAttempts,
		lastErr,
	)
}
