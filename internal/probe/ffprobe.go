package probe

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"video-processing-worker/internal/model"
)

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType    string            `json:"codec_type"`
	CodecName    string            `json:"codec_name"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	AvgFrameRate string            `json:"avg_frame_rate"`
	BitRate      string            `json:"bit_rate"`
	Duration     string            `json:"duration"`
	Channels     int               `json:"channels"`
	SampleRate   string            `json:"sample_rate"`
	SideDataList []ffprobeSideData `json:"side_data_list"`
}

type ffprobeSideData struct {
	SideDataType string `json:"side_data_type"`
	Rotation     int    `json:"rotation"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

func Run(source string) (model.ProbeResult, error) {

	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		source,
	)

	output, err := cmd.Output()
	if err != nil {
		return model.ProbeResult{}, fmt.Errorf("execute ffprobe: %w", err)
	}

	var data ffprobeOutput

	if err := json.Unmarshal(output, &data); err != nil {
		return model.ProbeResult{}, fmt.Errorf("parse ffprobe output: %w", err)
	}

	var videoStream *ffprobeStream
	var audioStream *ffprobeStream

	for i := range data.Streams {

		stream := &data.Streams[i]

		if stream.CodecType == "video" && videoStream == nil {
			videoStream = stream
		}

		if stream.CodecType == "audio" && audioStream == nil {
			audioStream = stream
		}
	}

	if videoStream == nil {
		return model.ProbeResult{}, fmt.Errorf("no video stream found")
	}

	durationSeconds := parseFloat(videoStream.Duration)

	if durationSeconds == 0 {
		durationSeconds = parseFloat(data.Format.Duration)
	}

	bitrate := parseInt(videoStream.BitRate)

	if bitrate == 0 {
		bitrate = parseInt(data.Format.BitRate)
	}

	rotation := getRotation(videoStream)

	displayWidth := videoStream.Width
	displayHeight := videoStream.Height

	if isQuarterTurn(rotation) {
		displayWidth = videoStream.Height
		displayHeight = videoStream.Width
	}

	var audioCodec string
	var audioBitrate int64
	var audioChannels int
	var audioSampleRate int

	if audioStream != nil {
		audioCodec = audioStream.CodecName
		audioBitrate = parseInt(audioStream.BitRate)
		audioChannels = audioStream.Channels
		audioSampleRate = int(parseInt(audioStream.SampleRate))
	}

	return model.ProbeResult{
		Width:         videoStream.Width,
		Height:        videoStream.Height,
		DisplayWidth:  displayWidth,
		DisplayHeight: displayHeight,
		Rotation:      rotation,
		DurationMs:    int64(durationSeconds * 1000),
		Codec:         videoStream.CodecName,
		BitrateBps:    bitrate,
		FrameRate:     parseFrameRate(videoStream.AvgFrameRate),

		HasAudio:        audioStream != nil,
		AudioCodec:      audioCodec,
		AudioBitrateBps: audioBitrate,
		AudioChannels:   audioChannels,
		AudioSampleRate: audioSampleRate,
	}, nil
}

func getRotation(stream *ffprobeStream) int {

	for _, sideData := range stream.SideDataList {

		if sideData.SideDataType == "Display Matrix" {
			return sideData.Rotation
		}
	}

	return 0
}

func isQuarterTurn(rotation int) bool {

	normalized := ((rotation % 360) + 360) % 360

	return normalized == 90 || normalized == 270
}

func parseFrameRate(value string) float64 {

	parts := strings.Split(value, "/")

	if len(parts) != 2 {
		return parseFloat(value)
	}

	numerator := parseFloat(parts[0])
	denominator := parseFloat(parts[1])

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func parseFloat(value string) float64 {

	if value == "" {
		return 0
	}

	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	return result
}

func parseInt(value string) int64 {

	if value == "" {
		return 0
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}

	return result
}
