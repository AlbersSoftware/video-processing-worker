package main

import (
	"encoding/json"
	"fmt"
	"os"

	"video-processing-worker/internal/model"
)

const videoProcessingJobEnv = "VIDEO_PROCESSING_JOB_JSON"

type sqsMessageEnvelope struct {
	Body json.RawMessage `json:"body"`
}

type sqsRecordsEnvelope struct {
	Records []struct {
		Body json.RawMessage `json:"body"`
	} `json:"Records"`
}

func loadJob(
	args []string,
) (model.VideoProcessingJob, error) {

	jobJSON := os.Getenv(
		videoProcessingJobEnv,
	)

	if jobJSON != "" {
		return loadJobFromJSON(
			[]byte(jobJSON),
		)
	}

	if len(args) < 2 {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"video processing job not provided; set %s or provide <job.json>",
			videoProcessingJobEnv,
		)
	}

	return loadJobFromFile(
		args[1],
	)
}

func loadJobFromFile(
	jobFile string,
) (model.VideoProcessingJob, error) {

	data, err := os.ReadFile(
		jobFile,
	)

	if err != nil {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"read job file %s: %w",
			jobFile,
			err,
		)
	}

	return loadJobFromJSON(
		data,
	)
}

func loadJobFromJSON(
	data []byte,
) (model.VideoProcessingJob, error) {

	job, err := loadDirectJob(
		data,
	)

	if err == nil {
		return job, nil
	}

	job, envelopeErr := loadJobFromSQSEnvelope(
		data,
	)

	if envelopeErr == nil {
		return job, nil
	}

	return model.VideoProcessingJob{}, fmt.Errorf(
		"unable to parse video processing job directly or from SQS envelope: direct=%v; envelope=%v",
		err,
		envelopeErr,
	)
}

func loadDirectJob(
	data []byte,
) (model.VideoProcessingJob, error) {

	var job model.VideoProcessingJob

	if err := json.Unmarshal(
		data,
		&job,
	); err != nil {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"parse direct video processing job: %w",
			err,
		)
	}

	if err := validateJob(
		job,
	); err != nil {
		return model.VideoProcessingJob{}, err
	}

	return job, nil
}

func loadJobFromSQSEnvelope(
	data []byte,
) (model.VideoProcessingJob, error) {

	job, err := loadJobFromSQSBodyEnvelope(
		data,
	)

	if err == nil {
		return job, nil
	}

	job, recordsErr := loadJobFromSQSRecordsEnvelope(
		data,
	)

	if recordsErr == nil {
		return job, nil
	}

	return model.VideoProcessingJob{}, fmt.Errorf(
		"unsupported SQS envelope: body=%v; records=%v",
		err,
		recordsErr,
	)
}

func loadJobFromSQSBodyEnvelope(
	data []byte,
) (model.VideoProcessingJob, error) {

	var envelope sqsMessageEnvelope

	if err := json.Unmarshal(
		data,
		&envelope,
	); err != nil {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"parse SQS body envelope: %w",
			err,
		)
	}

	if len(envelope.Body) == 0 {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"SQS body is missing",
		)
	}

	return loadJobFromSQSBody(
		envelope.Body,
	)
}

func loadJobFromSQSRecordsEnvelope(
	data []byte,
) (model.VideoProcessingJob, error) {

	var envelope sqsRecordsEnvelope

	if err := json.Unmarshal(
		data,
		&envelope,
	); err != nil {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"parse SQS records envelope: %w",
			err,
		)
	}

	if len(envelope.Records) == 0 {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"SQS records envelope contains no records",
		)
	}

	if len(envelope.Records) > 1 {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"expected one SQS record but received %d",
			len(envelope.Records),
		)
	}

	if len(envelope.Records[0].Body) == 0 {
		return model.VideoProcessingJob{}, fmt.Errorf(
			"SQS record body is missing",
		)
	}

	return loadJobFromSQSBody(
		envelope.Records[0].Body,
	)
}

func loadJobFromSQSBody(
	body json.RawMessage,
) (model.VideoProcessingJob, error) {

	var bodyString string

	if err := json.Unmarshal(
		body,
		&bodyString,
	); err == nil {

		return loadDirectJob(
			[]byte(bodyString),
		)
	}

	return loadDirectJob(
		body,
	)
}

func validateJob(
	job model.VideoProcessingJob,
) error {

	if job.MediaID == "" {
		return fmt.Errorf(
			"mediaId is required",
		)
	}

	if job.ProcessingJobID == "" {
		return fmt.Errorf(
			"processingJobId is required",
		)
	}

	if job.ProcessingGeneration < 1 {
		return fmt.Errorf(
			"processingGeneration must be greater than zero",
		)
	}

	if job.Source == "" {

		if job.SourceBucket == "" {
			return fmt.Errorf(
				"sourceBucket is required when source is not provided",
			)
		}

		if job.SourceStorageKey == "" {
			return fmt.Errorf(
				"sourceStorageKey is required when source is not provided",
			)
		}
	}

	if job.DerivedBucket == "" {
		return fmt.Errorf(
			"derivedBucket is required",
		)
	}

	if job.DerivedStoragePrefix == "" {
		return fmt.Errorf(
			"derivedStoragePrefix is required",
		)
	}

	return nil
}
