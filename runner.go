package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"video-processing-worker/internal/model"
	packaging "video-processing-worker/internal/package"
	"video-processing-worker/internal/probe"
	"video-processing-worker/internal/rendition"
	"video-processing-worker/internal/transcode"
)

func runJob(
	ctx context.Context,
	job model.VideoProcessingJob,
) (model.VideoProcessingResult, error) {

	fmt.Printf(
		"starting video processing job %s for media %s generation %d\n",
		job.ProcessingJobID,
		job.MediaID,
		job.ProcessingGeneration,
	)

	source, cleanup, err := resolveSource(
		ctx,
		job,
	)

	if err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"resolve video source: %w",
			err,
		)
	}

	defer cleanup()

	probeResult, err := probe.Run(
		source,
	)

	if err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"ffprobe: %w",
			err,
		)
	}

	fmt.Println("\nprobe result:")

	if err := printJSON(
		probeResult,
	); err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"serialize probe result: %w",
			err,
		)
	}

	plans := rendition.PlanForSource(
		probeResult,
	)

	fmt.Println("\nrendition plan:")

	if err := printJSON(
		plans,
	); err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"serialize rendition plan: %w",
			err,
		)
	}

	if len(plans) == 0 {

		fmt.Printf(
			"\nno video renditions needed for media %s; source short edge is %d\n",
			job.MediaID,
			min(
				probeResult.DisplayWidth,
				probeResult.DisplayHeight,
			),
		)

		return model.VideoProcessingResult{
			Successful: true,
			Renditions: []model.VideoRenditionResult{},
			Manifests:  []model.VideoManifestResult{},
		}, nil
	}

	outputRoot := filepath.Join(
		"output",
		job.MediaID,
		fmt.Sprintf(
			"generation-%d",
			job.ProcessingGeneration,
		),
	)

	videoOutputs, err := transcode.Run(
		source,
		probeResult,
		plans,
		outputRoot,
	)

	if err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"video transcode: %w",
			err,
		)
	}

	fmt.Println("\nvideo transcode outputs:")

	if err := printJSON(
		videoOutputs,
	); err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"serialize video transcode outputs: %w",
			err,
		)
	}

	audioOutput, err := runAudio(
		source,
		outputRoot,
		probeResult.HasAudio,
	)

	if err != nil {
		return model.VideoProcessingResult{}, err
	}

	packageOutput, err := packaging.Run(
		videoOutputs,
		audioOutput,
		outputRoot,
	)

	if err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"shaka packaging: %w",
			err,
		)
	}

	fmt.Println("\npackage output:")

	if err := printJSON(
		packageOutput,
	); err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"serialize package output: %w",
			err,
		)
	}

	if err := uploadPackage(
		ctx,
		job,
		outputRoot,
	); err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"upload packaged video output: %w",
			err,
		)
	}

	result, err := buildProcessingResult(
		job,
		probeResult,
		plans,
		videoOutputs,
		audioOutput,
		packageOutput,
	)

	if err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"build video processing result: %w",
			err,
		)
	}

	fmt.Println("\nvideo processing result:")

	if err := printJSON(
		result,
	); err != nil {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"serialize video processing result: %w",
			err,
		)
	}

	fmt.Println("\nvideo processing stage complete")

	return result, nil
}

func buildProcessingResult(
	job model.VideoProcessingJob,
	probeResult model.ProbeResult,
	plans []rendition.Plan,
	videoOutputs []transcode.Output,
	audioOutput *transcode.AudioOutput,
	packageOutput packaging.Output,
) (model.VideoProcessingResult, error) {

	planByName := make(
		map[string]rendition.Plan,
		len(plans),
	)

	for _, plan := range plans {
		planByName[plan.Name] = plan
	}

	renditions := make(
		[]model.VideoRenditionResult,
		0,
		len(videoOutputs)+1,
	)

	for _, output := range videoOutputs {

		plan, exists := planByName[output.Name]

		if !exists {
			return model.VideoProcessingResult{}, fmt.Errorf(
				"rendition plan not found for output %s",
				output.Name,
			)
		}

		targetBitrateBps, err := parseBitrateBps(
			plan.VideoBitrate,
		)

		if err != nil {
			return model.VideoProcessingResult{}, fmt.Errorf(
				"parse video bitrate for rendition %s: %w",
				output.Name,
				err,
			)
		}

		width := output.Width
		height := output.Height
		frameRate := probeResult.FrameRate

		renditions = append(
			renditions,
			model.VideoRenditionResult{
				StreamType:       "VIDEO",
				Codec:            "H264",
				Container:        "mp4",
				Width:            &width,
				Height:           &height,
				FrameRate:        &frameRate,
				TargetBitrateBps: &targetBitrateBps,
				StoragePrefix: buildDerivedStoragePrefix(
					job.DerivedStoragePrefix,
					"package",
					"video",
					output.Name,
				),
				Status: "READY",
			},
		)
	}

	if audioOutput != nil {

		audioBitrateBps, err := parseBitrateBps(
			audioOutput.Bitrate,
		)

		if err != nil {
			return model.VideoProcessingResult{}, fmt.Errorf(
				"parse audio bitrate: %w",
				err,
			)
		}

		renditions = append(
			renditions,
			model.VideoRenditionResult{
				StreamType:       "AUDIO",
				Codec:            "AAC",
				Container:        "mp4",
				TargetBitrateBps: &audioBitrateBps,
				StoragePrefix: buildDerivedStoragePrefix(
					job.DerivedStoragePrefix,
					"package",
					"audio",
					"aac",
				),
				Status: "READY",
			},
		)
	}

	if packageOutput.HLSMasterPlaylist == "" {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"HLS master playlist was not produced",
		)
	}

	if packageOutput.DASHManifest == "" {
		return model.VideoProcessingResult{}, fmt.Errorf(
			"DASH manifest was not produced",
		)
	}

	manifests := []model.VideoManifestResult{
		{
			ManifestType: "HLS",
			StorageKey: buildDerivedStorageKey(
				job.DerivedStoragePrefix,
				"master.m3u8",
			),
		},
		{
			ManifestType: "DASH",
			StorageKey: buildDerivedStorageKey(
				job.DerivedStoragePrefix,
				"manifest.mpd",
			),
		},
	}

	return model.VideoProcessingResult{
		Successful: true,
		Renditions: renditions,
		Manifests:  manifests,
	}, nil
}

func buildDerivedStoragePrefix(
	derivedStoragePrefix string,
	parts ...string,
) string {

	allParts := []string{
		strings.TrimSuffix(
			derivedStoragePrefix,
			"/",
		),
	}

	allParts = append(
		allParts,
		parts...,
	)

	return path.Join(
		allParts...,
	) + "/"
}

func parseBitrateBps(
	bitrate string,
) (int64, error) {

	value := strings.TrimSpace(
		strings.ToLower(bitrate),
	)

	if value == "" {
		return 0, fmt.Errorf(
			"bitrate is empty",
		)
	}

	multiplier := int64(1)

	switch {

	case strings.HasSuffix(
		value,
		"k",
	):
		multiplier = 1000
		value = strings.TrimSuffix(
			value,
			"k",
		)

	case strings.HasSuffix(
		value,
		"m",
	):
		multiplier = 1000000
		value = strings.TrimSuffix(
			value,
			"m",
		)
	}

	numericValue, err := strconv.ParseInt(
		value,
		10,
		64,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"invalid bitrate %q: %w",
			bitrate,
			err,
		)
	}

	return numericValue * multiplier, nil
}

func runAudio(
	source string,
	outputRoot string,
	hasAudio bool,
) (*transcode.AudioOutput, error) {

	if !hasAudio {
		fmt.Println(
			"\nsource contains no audio stream; skipping audio transcode",
		)

		return nil, nil
	}

	output, err := transcode.RunAudio(
		source,
		outputRoot,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"audio transcode: %w",
			err,
		)
	}

	fmt.Println("\naudio output:")

	if err := printJSON(
		output,
	); err != nil {
		return nil, fmt.Errorf(
			"serialize audio output: %w",
			err,
		)
	}

	return &output, nil
}

func printJSON(
	value any,
) error {

	data, err := json.MarshalIndent(
		value,
		"",
		"  ",
	)

	if err != nil {
		return err
	}

	fmt.Println(
		string(data),
	)

	return nil
}
