package model

type ProbeResult struct {
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	DisplayWidth  int     `json:"displayWidth"`
	DisplayHeight int     `json:"displayHeight"`
	Rotation      int     `json:"rotation"`
	DurationMs    int64   `json:"durationMs"`
	Codec         string  `json:"codec"`
	BitrateBps    int64   `json:"bitrateBps"`
	FrameRate     float64 `json:"frameRate"`

	HasAudio        bool   `json:"hasAudio"`
	AudioCodec      string `json:"audioCodec,omitempty"`
	AudioBitrateBps int64  `json:"audioBitrateBps,omitempty"`
	AudioChannels   int    `json:"audioChannels,omitempty"`
	AudioSampleRate int    `json:"audioSampleRate,omitempty"`
}
