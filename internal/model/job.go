package model

type VideoProcessingJob struct {
	MediaID              string `json:"mediaId"`
	ProcessingJobID      string `json:"processingJobId"`
	ProcessingGeneration int    `json:"processingGeneration"`

	Source               string `json:"source,omitempty"`
	SourceBucket         string `json:"sourceBucket,omitempty"`
	SourceStorageKey     string `json:"sourceStorageKey,omitempty"`
	DerivedBucket        string `json:"derivedBucket,omitempty"`
	DerivedStoragePrefix string `json:"derivedStoragePrefix"`
}
