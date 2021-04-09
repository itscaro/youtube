package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var downloadCmdOpts struct {
	autoMode string
}

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:     "download",
	Short:   "Downloads a video from youtube",
	Example: `download https://www.youtube.com/watch\?v\=XbNghLqsVwU https://www.youtube.com/watch\?v\=XbNghLqsVwU`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    download,
}

var (
	ffmpegCheck            error
	ffmpegCheckInitialized bool
	outputFile             string
	outputDir              string
)

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVarP(&outputDir, "directory", "d", ".", "The output directory.")
	downloadCmd.Flags().StringVarP(&downloadCmdOpts.autoMode, "mode", "m", "auto", "(none|auto): auto download video & audio stream and join them")
	addQualityFlag(downloadCmd.Flags())
	addCodecFlag(downloadCmd.Flags())
}

func download(cmd *cobra.Command, args []string) error {
	log.Println("download to directory", outputDir)

	downloadSeparatedStreams := strings.HasPrefix(outputQuality, "hd") || downloadCmdOpts.autoMode == "auto"
	if downloadSeparatedStreams {
		if err := checkFFMPEG(); err != nil {
			return err
		}
	}

	var errors []string
	for _, videoURL := range args {
		video, format, err := getVideoWithFormat(videoURL)
		if err != nil {
			log.Printf("⛔ %s: '%s'\n", videoURL, err)
			errors = append(errors, err.Error())
			continue
		}
		log.Printf("▶ %s: '%s'\n", video.ID, video.Title)

		if downloadSeparatedStreams {
			if err := downloader.DownloadWithHighQuality(context.Background(), outputFile, video, outputQuality); err != nil {
				errors = append(errors, err.Error())
			}
		} else if err := downloader.Download(context.Background(), video, format, outputFile); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("failure to process videos:\n" + strings.Join(errors, "\n"))
	}
	return nil
}

func checkFFMPEG() error {
	if !ffmpegCheckInitialized {
		fmt.Println("check ffmpeg is installed....")
		if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
			ffmpegCheck = fmt.Errorf("please check ffmpegCheck is installed correctly")
		}
	}

	return ffmpegCheck
}
