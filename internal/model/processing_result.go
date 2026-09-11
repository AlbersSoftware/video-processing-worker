package model

type VideoProcessingResult struct {
	Successful     bool                   `json:"successful"`
	FailureMessage *string                `json:"failureMessage"`
	Renditions     []VideoRenditionResult `json:"renditions"`
	Manifests      []VideoManifestResult  `json:"manifests"`
}

type VideoRenditionResult struct {
	StreamType         string   `json:"streamType"`
	Codec              string   `json:"codec"`
	Container          string   `json:"container"`
	Width              *int     `json:"width"`
	Height             *int     `json:"height"`
	FrameRate          *float64 `json:"frameRate"`
	TargetBitrateBps   *int64   `json:"targetBitrateBps"`
	AverageBitrateBps  *int64   `json:"averageBitrateBps"`
	PeakBitrateBps     *int64   `json:"peakBitrateBps"`
	ManifestBitrateBps *int64   `json:"manifestBitrateBps"`
	CodecProfile       *string  `json:"codecProfile"`
	CodecLevel         *string  `json:"codecLevel"`
	StoragePrefix      string   `json:"storagePrefix"`
	Status             string   `json:"status"`
}

type VideoManifestResult struct {
	ManifestType string `json:"manifestType"`
	StorageKey   string `json:"storageKey"`
}
