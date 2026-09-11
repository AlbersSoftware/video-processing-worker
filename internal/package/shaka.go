package packaging

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"video-processing-worker/internal/transcode"
)

type Output struct {
	HLSMasterPlaylist string `json:"hlsMasterPlaylist"`
	DASHManifest      string `json:"dashManifest"`
}

func Run(videoOutputs []transcode.Output,
	audioOutput *transcode.AudioOutput,
	outputRoot string) (Output, error) {

	if len(videoOutputs) == 0 {
		return Output{}, fmt.Errorf(
			"at least one video rendition is required for packaging",
		)
	}

	packageRoot := filepath.Join(
		outputRoot,
		"package",
	)

	if err := os.MkdirAll(packageRoot, 0755); err != nil {
		return Output{}, fmt.Errorf(
			"create package output directory: %w",
			err,
		)
	}

	var args []string

	for _, video := range videoOutputs {

		outputDirectory := filepath.Join(
			packageRoot,
			"video",
			video.Name,
		)

		if err := os.MkdirAll(outputDirectory, 0755); err != nil {
			return Output{}, fmt.Errorf(
				"create video package directory %s: %w",
				video.Name,
				err,
			)
		}

		descriptor := fmt.Sprintf(
			"in=%s,stream=video,init_segment=%s,segment_template=%s,playlist_name=%s",
			video.FilePath,
			filepath.Join(
				outputDirectory,
				"init.mp4",
			),
			filepath.Join(
				outputDirectory,
				"$Number$.m4s",
			),
			filepath.Join(
				"video",
				video.Name,
				"main.m3u8",
			),
		)

		args = append(
			args,
			descriptor,
		)
	}

	if audioOutput != nil {

		audioDirectory := filepath.Join(
			packageRoot,
			"audio",
			"aac",
		)

		if err := os.MkdirAll(audioDirectory, 0755); err != nil {
			return Output{}, fmt.Errorf(
				"create audio package directory: %w",
				err,
			)
		}

		audioDescriptor := fmt.Sprintf(
			"in=%s,stream=audio,init_segment=%s,segment_template=%s,playlist_name=%s,hls_group_id=audio,hls_name=Main",
			audioOutput.FilePath,
			filepath.Join(
				audioDirectory,
				"init.mp4",
			),
			filepath.Join(
				audioDirectory,
				"$Number$.m4s",
			),
			filepath.Join(
				"audio",
				"aac",
				"main.m3u8",
			),
		)

		args = append(
			args,
			audioDescriptor,
		)
	}

	hlsMaster := filepath.Join(
		packageRoot,
		"master.m3u8",
	)

	dashManifest := filepath.Join(
		packageRoot,
		"manifest.mpd",
	)

	args = append(
		args,
		"--hls_master_playlist_output",
		hlsMaster,
		"--mpd_output",
		dashManifest,
	)

	if audioOutput == nil {
		fmt.Println("packaging video-only HLS and DASH")
	} else {
		fmt.Println("packaging HLS and DASH")
	}

	cmd := exec.Command(
		"packager",
		args...,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return Output{}, fmt.Errorf(
			"package HLS/DASH: %w",
			err,
		)
	}

	return Output{
		HLSMasterPlaylist: hlsMaster,
		DASHManifest:      dashManifest,
	}, nil
}
