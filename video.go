package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Video struct {
	ID                string
	Title             string
	Description       string
	Author            string
	Duration          time.Duration
	Formats           FormatList
	AudioFormats      FormatList
	VideoFormats      FormatList
	VideoAudioFormats FormatList
	Thumbnails        Thumbnails
	DASHManifestURL   string // URI of the DASH manifest file
	HLSManifestURL    string // URI of the HLS manifest file
}

func (v *Video) parseVideoInfo(body []byte) error {
	answer, err := url.ParseQuery(string(body))
	if err != nil {
		return err
	}

	status := answer.Get("status")
	if status != "ok" {
		return &ErrResponseStatus{
			Status: status,
			Reason: answer.Get("reason"),
		}
	}

	// read the streams map
	playerResponse := answer.Get("player_response")
	if playerResponse == "" {
		return errors.New("no player_response found in the server's answer")
	}

	var prData playerResponseData
	if err := json.Unmarshal([]byte(playerResponse), &prData); err != nil {
		return fmt.Errorf("unable to parse player response JSON: %w", err)
	}

	if err := v.isVideoFromInfoDownloadable(prData); err != nil {
		return err
	}

	return v.extractDataFromPlayerResponse(prData)
}

func (v *Video) isVideoFromInfoDownloadable(prData playerResponseData) error {
	return v.isVideoDownloadable(prData, false)
}

var playerResponsePattern = regexp.MustCompile(`var ytInitialPlayerResponse\s*=\s*(\{.+?\});`)

func (v *Video) parseVideoPage(body []byte) error {
	initialPlayerResponse := playerResponsePattern.FindSubmatch(body)
	if initialPlayerResponse == nil || len(initialPlayerResponse) < 2 {
		return errors.New("no ytInitialPlayerResponse found in the server's answer")
	}

	var prData playerResponseData
	if err := json.Unmarshal(initialPlayerResponse[1], &prData); err != nil {
		return fmt.Errorf("unable to parse player response JSON: %w", err)
	}

	if err := v.isVideoFromPageDownloadable(prData); err != nil {
		return err
	}

	return v.extractDataFromPlayerResponse(prData)
}

func (v *Video) isVideoFromPageDownloadable(prData playerResponseData) error {
	return v.isVideoDownloadable(prData, true)
}

func (v *Video) isVideoDownloadable(prData playerResponseData, isVideoPage bool) error {
	// Check if video is downloadable
	if prData.PlayabilityStatus.Status == "OK" {
		return nil
	}

	if !isVideoPage && !prData.PlayabilityStatus.PlayableInEmbed {
		return ErrNotPlayableInEmbed
	}

	return &ErrPlayabiltyStatus{
		Status: prData.PlayabilityStatus.Status,
		Reason: prData.PlayabilityStatus.Reason,
	}
}

func (v *Video) extractDataFromPlayerResponse(prData playerResponseData) error {
	v.Title = prData.VideoDetails.Title
	v.Description = prData.VideoDetails.ShortDescription
	v.Author = prData.VideoDetails.Author
	v.Thumbnails = prData.VideoDetails.Thumbnail.Thumbnails

	if seconds, _ := strconv.Atoi(prData.Microformat.PlayerMicroformatRenderer.LengthSeconds); seconds > 0 {
		v.Duration = time.Duration(seconds) * time.Second
	}

	// Assign Streams
	v.Formats = append(prData.StreamingData.Formats, prData.StreamingData.AdaptiveFormats...)
	for _, f := range v.Formats {
		if strings.HasPrefix(f.MimeType, "video") {
			if f.AudioChannels == 0 {
				v.VideoFormats = append(v.VideoFormats, f)
			} else {
				v.VideoAudioFormats = append(v.VideoAudioFormats, f)
			}
		} else if strings.HasPrefix(f.MimeType, "audio") {
			v.AudioFormats = append(v.AudioFormats, f)
		}
	}

	if len(v.Formats) == 0 {
		return errors.New("no formats found in the server's answer")
	}
	v.SortFormats()

	v.HLSManifestURL = prData.StreamingData.HlsManifestURL
	v.DASHManifestURL = prData.StreamingData.DashManifestURL

	return nil
}

func (v *Video) FilterCodec(codec []string) {
	v.Formats = v.Formats.FilterCodec(codec)
	v.VideoAudioFormats = v.VideoAudioFormats.FilterCodec(codec)
	v.AudioFormats = v.AudioFormats.FilterCodec(codec)
	v.VideoFormats = v.VideoFormats.FilterCodec(codec)
	v.SortFormats()
}

func (v *Video) FilterQuality(quality []string) {
	v.Formats = v.Formats.FilterQuality(quality)
	v.VideoAudioFormats = v.VideoAudioFormats.FilterQuality(quality)
	v.AudioFormats = v.AudioFormats.FilterQuality(quality)
	v.VideoFormats = v.VideoFormats.FilterQuality(quality)
	v.SortFormats()
}

// SortFormats sort all Formats fields
func (v *Video) SortFormats() {
	sort.SliceStable(v.Formats, func(i, j int) bool {
		return SortFormat(i, j, v.Formats)
	})
	sort.SliceStable(v.VideoAudioFormats, func(i, j int) bool {
		return SortFormat(i, j, v.VideoAudioFormats)
	})
	sort.SliceStable(v.AudioFormats, func(i, j int) bool {
		return SortFormat(i, j, v.AudioFormats)
	})
	sort.SliceStable(v.VideoFormats, func(i, j int) bool {
		return SortFormat(i, j, v.VideoFormats)
	})
}

// SortFormat sorts video by resolution, FPS, codec (av01, vp9, avc1), bitrate
// sorts audio by codec (mp4, opus), channels, bitrate, sample rate
func SortFormat(i int, j int, formats FormatList) bool {
	// Sort by Width
	if formats[i].Width == formats[j].Width {
		// Sort by FPS
		if formats[i].FPS == formats[j].FPS {
			if formats[i].FPS == 0 && formats[i].AudioChannels > 0 && formats[j].AudioChannels > 0 {
				// Audio
				// Sort by codec
				codec := map[int]int{}
				for _, index := range []int{i, j} {
					if strings.Contains(formats[index].MimeType, "mp4") {
						codec[index] = 1
					} else if strings.Contains(formats[index].MimeType, "opus") {
						codec[index] = 2
					}
				}
				if codec[i] == codec[j] {
					// Sort by Audio Channel
					if formats[i].AudioChannels == formats[j].AudioChannels {
						// Sort by Audio Bitrate
						if formats[i].Bitrate == formats[j].Bitrate {
							// Sort by Audio Sample Rate
							return formats[i].AudioSampleRate > formats[j].AudioSampleRate
						}
						return formats[i].Bitrate > formats[j].Bitrate
					}
					return formats[i].AudioChannels > formats[j].AudioChannels
				}
				return codec[i] < codec[j]
			} else {
				// Video
				// Sort by codec
				codec := map[int]int{}
				for _, index := range []int{i, j} {
					if strings.Contains(formats[index].MimeType, "av01") {
						codec[index] = 1
					} else if strings.Contains(formats[index].MimeType, "vp9") {
						codec[index] = 2
					} else if strings.Contains(formats[index].MimeType, "avc1") {
						codec[index] = 3
					}
				}
				if codec[i] == codec[j] {
					// Sort by Audio Bitrate
					return formats[i].Bitrate > formats[j].Bitrate
				}
				return codec[i] < codec[j]
			}
		}
		return formats[i].FPS > formats[j].FPS
	}
	return formats[i].Width > formats[j].Width
}
