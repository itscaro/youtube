package main

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:     "download",
	Short:   "Downloads a video from youtube",
	Example: `download -o "Campaign Diary".mp4 https://www.youtube.com/watch\?v\=XbNghLqsVwU`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return download(args[0])
	},
}

var (
	ffmpegCheck            error
	ffmpegCheckInitialized bool
	outputFile             string
	outputDir              string
)

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVarP(&outputFile, "filename", "o", "", "The output file, the default is genated by the video title.")
	downloadCmd.Flags().StringVarP(&outputDir, "directory", "d", ".", "The output directory.")
	addQualityFlag(downloadCmd.Flags())
	addCodecFlag(downloadCmd.Flags())
}

func download(id string) error {
	video, format, err := getVideoWithFormat(id)
	if err != nil {
		return err
	}

	log.Println("download to directory", outputDir)

	if outputQuality == "hd1080" {
		if err := checkFFMPEG(); err != nil {
			return err
		}

		return downloader.DownloadWithHighQuality(context.Background(), outputFile, video, outputQuality)
	}

	return downloader.Download(context.Background(), video, format, outputFile)
}
