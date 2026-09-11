package transcode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type AudioOutput struct {
	Codec      string `json:"codec"`
	Bitrate    string `json:"bitrate"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
	FilePath   string `json:"filePath"`
}

func RunAudio(source string, outputRoot string) (AudioOutput, error) {

	outputDirectory := filepath.Join(
		outputRoot,
		"audio",
		"aac",
	)

	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return AudioOutput{}, fmt.Errorf(
			"create audio output directory: %w",
			err,
		)
	}

	outputFile := filepath.Join(
		outputDirectory,
		"audio.m4a",
	)

	fmt.Println("transcoding shared AAC audio rendition")

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", source,

		"-map", "0:a:0",

		"-vn",

		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "48000",
		"-ac", "2",

		outputFile,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return AudioOutput{}, fmt.Errorf(
			"transcode audio rendition: %w",
			err,
		)
	}

	return AudioOutput{
		Codec:      "aac",
		Bitrate:    "128k",
		SampleRate: 48000,
		Channels:   2,
		FilePath:   outputFile,
	}, nil
}
