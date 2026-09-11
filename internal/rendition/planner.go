package rendition

import "video-processing-worker/internal/model"

type Plan struct {
	Name         string `json:"name"`
	ShortEdge    int    `json:"shortEdge"`
	VideoBitrate string `json:"videoBitrate"`
	MaxRate      string `json:"maxRate"`
	BufferSize   string `json:"bufferSize"`
}

func PlanForSource(probe model.ProbeResult) []Plan {

	candidates := []Plan{
		{
			Name:         "2160p",
			ShortEdge:    2160,
			VideoBitrate: "12000k",
			MaxRate:      "14000k",
			BufferSize:   "24000k",
		},
		{
			Name:         "1440p",
			ShortEdge:    1440,
			VideoBitrate: "8000k",
			MaxRate:      "9000k",
			BufferSize:   "16000k",
		},
		{
			Name:         "1080p",
			ShortEdge:    1080,
			VideoBitrate: "5000k",
			MaxRate:      "5500k",
			BufferSize:   "10000k",
		},
		{
			Name:         "720p",
			ShortEdge:    720,
			VideoBitrate: "2800k",
			MaxRate:      "3200k",
			BufferSize:   "5600k",
		},
		{
			Name:         "480p",
			ShortEdge:    480,
			VideoBitrate: "1400k",
			MaxRate:      "1600k",
			BufferSize:   "2800k",
		},
		{
			Name:         "360p",
			ShortEdge:    360,
			VideoBitrate: "800k",
			MaxRate:      "900k",
			BufferSize:   "1600k",
		},
	}

	sourceShortEdge := min(
		probe.DisplayWidth,
		probe.DisplayHeight,
	)

	var plans []Plan

	for _, candidate := range candidates {

		// Never upscale beyond the source's displayed short edge.
		if candidate.ShortEdge > sourceShortEdge {
			continue
		}

		plans = append(plans, candidate)
	}

	return plans
}
