package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type ProbeResult struct {
	Duration float64
	Width    int
	Height   int
	Codec    string
	Bitrate  int64
}

func Probe(path string, ffprobeBinary string) (*ProbeResult, error) {
	if ffprobeBinary == "" {
		ffprobeBinary = "ffprobe"
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	}

	cmd := exec.Command(ffprobeBinary, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	probe := &ProbeResult{}

	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			probe.Width = stream.Width
			probe.Height = stream.Height
			probe.Codec = stream.CodecName
		}
	}

	if result.Format.Duration != "" {
		d, err := strconv.ParseFloat(result.Format.Duration, 64)
		if err == nil {
			probe.Duration = d
		}
	}

	if result.Format.BitRate != "" {
		b, err := strconv.ParseInt(result.Format.BitRate, 10, 64)
		if err == nil {
			probe.Bitrate = b
		}
	}

	return probe, nil
}
