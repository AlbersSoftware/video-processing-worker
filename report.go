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

	client := &http.Client{
		Timeout: processingResultRequestTimeout,
	}

	response, err := client.Do(
		request,
	)

	if err != nil {
		return fmt.Errorf(
			"send video processing result: %w",
			err,
		)
	}

	defer response.Body.Close()

	responseBody, err := io.ReadAll(
		response.Body,
	)

	if err != nil {
		return fmt.Errorf(
			"read video processing result response: %w",
			err,
		)
	}

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {

		return fmt.Errorf(
			"media-svc rejected video processing result: status=%d body=%s",
			response.StatusCode,
			string(responseBody),
		)
	}

	fmt.Printf(
		"video processing result reported successfully for media %s processing job %s\n",
		job.MediaID,
		job.ProcessingJobID,
	)

	return nil
}
