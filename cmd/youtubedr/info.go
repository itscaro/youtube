package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kkdai/youtube/v2"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var infoCmdOpts struct {
	outputFormat string
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:          "info",
	Short:        "Print metadata of the desired videos",
	Example:      `info https://www.youtube.com/watch\?v\=XbNghLqsVwU https://www.youtube.com/watch\?v\=XbNghLqsVwU`,
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !strings.Contains("|full|media|media-split|media-csv|media-split-csv|", fmt.Sprintf("|%s|", infoCmdOpts.outputFormat)) {
			return fmt.Errorf("output format %s is not valid", infoCmdOpts.outputFormat)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		for _, videoURL := range args {
			video, err := getDownloader().GetVideo(videoURL)
			if err != nil {
				if infoCmdOpts.outputFormat == "media-csv" {
					fmt.Printf("-2,%s,%s\n", videoURL, err)
				} else {
					fmt.Printf("ERROR: %s - %s\n", videoURL, err)
				}
				continue
			}
			if len(video.Formats) == 0 {
				if infoCmdOpts.outputFormat == "media-csv" {
					fmt.Printf("-3,%s,%s\n", videoURL, "no formats found")
				} else {
					fmt.Printf("ERROR: %s - %s\n", videoURL, "no formats found")
				}
				continue
			}

			if len(codec) > 0 {
				video.FilterCodec(codec)
			}
			if len(quality) > 0 {
				video.FilterQuality(quality)
			}

			data := buildFormats(video)

			if strings.HasSuffix(infoCmdOpts.outputFormat, "csv") {
				fmt.Printf("-,%s,%s\n", videoURL, video.Title)
				w := csv.NewWriter(os.Stdout)
				for _, record := range data {
					if err := w.Write(record); err != nil {
						fmt.Printf("-1,%s,%s\n", videoURL, err)
					}
				}
				w.Flush()
				if err := w.Error(); err != nil {
					fmt.Printf("-1,%s,%s\n", videoURL, err)
					continue
				}
			} else {
				fmt.Println("Title:      ", video.Title)
				if infoCmdOpts.outputFormat == "full" {
					fmt.Println("Author:     ", video.Author)
					fmt.Println("Duration:   ", video.Duration)
					fmt.Println("Description:", video.Description)
					fmt.Println()
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.SetAutoWrapText(false)
				table.SetHeader([]string{"itag", "fps", "video\nquality", "video\nquality\nlabel", "audio\nquality", "audio\n channels", "size [MB]", "bitrate", "sample\nrate", "MimeType"})
				table.AppendBulk(data)
				table.Render()
			}
		}
	},
}

func buildFormats(video *youtube.Video) (data [][]string) {
	if strings.HasPrefix(infoCmdOpts.outputFormat, "media-split") {
		for _, format := range video.VideoAudioFormats {
			bitrate := format.AverageBitrate
			if bitrate == 0 {
				// Some formats don't have the average bitrate
				bitrate = format.Bitrate
			}

			size, _ := strconv.ParseInt(format.ContentLength, 10, 64)
			if size == 0 {
				// Some formats don't have this information
				size = int64(float64(bitrate) * video.Duration.Seconds() / 8)
			}

			videoQuality := format.Quality
			// If no FPS then it's audio stream
			if format.FPS == 0 {
				videoQuality = ""
			}

			data = append(data, []string{
				strconv.Itoa(format.ItagNo),
				strconv.Itoa(format.FPS),
				videoQuality,
				format.QualityLabel,
				strings.ToLower(strings.TrimPrefix(format.AudioQuality, "AUDIO_QUALITY_")),
				strconv.Itoa(format.AudioChannels),
				fmt.Sprintf("%0.1f", float64(size)/1024/1024),
				strconv.Itoa(bitrate),
				format.AudioSampleRate,
				format.MimeType,
			})
		}

		data = append(data, []string{"-", "-", "-", "-", "-", "-", "-", "-", "-", "-"})

		for _, format := range video.VideoFormats {
			bitrate := format.AverageBitrate
			if bitrate == 0 {
				// Some formats don't have the average bitrate
				bitrate = format.Bitrate
			}

			size, _ := strconv.ParseInt(format.ContentLength, 10, 64)
			if size == 0 {
				// Some formats don't have this information
				size = int64(float64(bitrate) * video.Duration.Seconds() / 8)
			}

			videoQuality := format.Quality
			// If no FPS then it's audio stream
			if format.FPS == 0 {
				videoQuality = ""
			}

			data = append(data, []string{
				strconv.Itoa(format.ItagNo),
				strconv.Itoa(format.FPS),
				videoQuality,
				format.QualityLabel,
				strings.ToLower(strings.TrimPrefix(format.AudioQuality, "AUDIO_QUALITY_")),
				strconv.Itoa(format.AudioChannels),
				fmt.Sprintf("%0.1f", float64(size)/1024/1024),
				strconv.Itoa(bitrate),
				format.AudioSampleRate,
				format.MimeType,
			})
		}

		data = append(data, []string{"-", "-", "-", "-", "-", "-", "-", "-", "-", "-"})

		for _, format := range video.AudioFormats {
			bitrate := format.AverageBitrate
			if bitrate == 0 {
				// Some formats don't have the average bitrate
				bitrate = format.Bitrate
			}

			size, _ := strconv.ParseInt(format.ContentLength, 10, 64)
			if size == 0 {
				// Some formats don't have this information
				size = int64(float64(bitrate) * video.Duration.Seconds() / 8)
			}

			videoQuality := format.Quality
			// If no FPS then it's audio stream
			if format.FPS == 0 {
				videoQuality = ""
			}

			data = append(data, []string{
				strconv.Itoa(format.ItagNo),
				strconv.Itoa(format.FPS),
				videoQuality,
				format.QualityLabel,
				strings.ToLower(strings.TrimPrefix(format.AudioQuality, "AUDIO_QUALITY_")),
				strconv.Itoa(format.AudioChannels),
				fmt.Sprintf("%0.1f", float64(size)/1024/1024),
				strconv.Itoa(bitrate),
				format.AudioSampleRate,
				format.MimeType,
			})
		}
	} else {
		for _, format := range video.Formats {
			bitrate := format.AverageBitrate
			if bitrate == 0 {
				// Some formats don't have the average bitrate
				bitrate = format.Bitrate
			}

			size, _ := strconv.ParseInt(format.ContentLength, 10, 64)
			if size == 0 {
				// Some formats don't have this information
				size = int64(float64(bitrate) * video.Duration.Seconds() / 8)
			}

			videoQuality := format.Quality
			// If no FPS then it's audio stream
			if format.FPS == 0 {
				videoQuality = ""
			}

			data = append(data, []string{
				strconv.Itoa(format.ItagNo),
				strconv.Itoa(format.FPS),
				videoQuality,
				format.QualityLabel,
				strings.ToLower(strings.TrimPrefix(format.AudioQuality, "AUDIO_QUALITY_")),
				strconv.Itoa(format.AudioChannels),
				fmt.Sprintf("%0.1f", float64(size)/1024/1024),
				strconv.Itoa(bitrate),
				format.AudioSampleRate,
				format.MimeType,
			})
		}
	}

	return
}

func init() {
	rootCmd.AddCommand(infoCmd)

	infoCmd.Flags().StringVarP(&infoCmdOpts.outputFormat, "output", "o", "full", "full, media, media-csv")
	addCodecFlag(infoCmd.Flags())
	addQualityArrayFlag(infoCmd.Flags())
}
