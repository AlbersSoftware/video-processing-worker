package transcode

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"video-processing-worker/internal/model"
	"video-processing-worker/internal/rendition"
)

type Output struct {
	Name     string `json:"name"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FilePath string `json:"filePath"`
}

func Run(source string, probe model.ProbeResult,
	plans []rendition.Plan, outputRoot string) ([]Output, error) {

	if err := os.MkdirAll(outputRoot, 0755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	var outputs []Output

	portrait := probe.DisplayHeight > probe.DisplayWidth

	for _, plan := range plans {

		outputDirectory := filepath.Join(
			outputRoot,
			"video",
			plan.Name,
		)

		if err := os.MkdirAll(outputDirectory, 0755); err != nil {
			return nil, fmt.Errorf(
				"create rendition directory %s: %w",
				plan.Name,
				err,
			)
		}

		outputFile := filepath.Join(
			outputDirectory,
			"video.mp4",
		)

		scaleFilter := buildScaleFilter(
			portrait,
			plan.ShortEdge,
		)

		fmt.Printf(
			"transcoding %s rendition with scale %s\n",
			plan.Name,
			scaleFilter,
		)

		cmd := exec.Command(
			"ffmpeg",
			"-y",
			"-i", source,

			"-map", "0:v:0",

			"-vf", scaleFilter,

			"-c:v", "libx264",
			"-preset", "medium",

			"-b:v", plan.VideoBitrate,
			"-maxrate", plan.MaxRate,
			"-bufsize", plan.BufferSize,

			"-pix_fmt", "yuv420p",

			"-an",

			outputFile,
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf(
				"transcode %s rendition: %w",
				plan.Name,
				err,
			)
		}

		width, height := expectedDimensions(
			probe,
			plan.ShortEdge,
		)

		outputs = append(outputs, Output{
			Name:     plan.Name,
			Width:    width,
			Height:   height,
			FilePath: outputFile,
		})
	}

	return outputs, nil
}

func buildScaleFilter(portrait bool, shortEdge int) string {

	if portrait {
		return fmt.Sprintf(
			"scale=%d:-2",
			shortEdge,
		)
	}

	return fmt.Sprintf(
		"scale=-2:%d",
		shortEdge,
	)
}

func expectedDimensions(probe model.ProbeResult,
	shortEdge int) (int, int) {

	displayWidth := probe.DisplayWidth
	displayHeight := probe.DisplayHeight

	if displayHeight > displayWidth {

		height := makeEven(
			float64(shortEdge) *
				float64(displayHeight) /
				float64(displayWidth),
		)

		return shortEdge, height
	}

	width := makeEven(
		float64(shortEdge) *
			float64(displayWidth) /
			float64(displayHeight),
	)

	return width, shortEdge
}

func makeEven(value float64) int {

	return int(
		math.Round(value/2) * 2,
	)
}
