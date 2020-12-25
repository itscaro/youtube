package downloader

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kkdai/youtube/v2"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

// Downloader offers high level functions to download videos into files
type Downloader struct {
	youtube.Client
	OutputDir string // optional directory to store the files
}

func (dl *Downloader) getOutputFile(v *youtube.Video, format *youtube.Format, outputFile string) (string, error) {
	if outputFile == "" {
		outputFile = SanitizeFilename(v.Title)
		outputFile += pickIdealFileExtension(format.MimeType)
	}

	if dl.OutputDir != "" {
		if err := os.MkdirAll(dl.OutputDir, 0o755); err != nil {
			return "", err
		}
		outputFile = filepath.Join(dl.OutputDir, outputFile)
	}

	return outputFile, nil
}

// Download : Starting download video by arguments.
func (dl *Downloader) Download(ctx context.Context, v *youtube.Video, format *youtube.Format, outputFile string) error {
	dl.LogInfo("Video '%s'", v.Title)
	dl.LogInfo("Video (Quality '%s' | FPS '%d' | Codec '%s') - Audio (Channels '%d')",
		format.QualityLabel, format.FPS, format.MimeType, format.AudioChannels)

	destFile, err := dl.getOutputFile(v, format, outputFile)
	if err != nil {
		return err
	}

	// Create output file
	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()

	dl.LogInfo("Download to file=%s", destFile)
	return dl.videoDLWorker(ctx, out, v, format)
}

// DownloadWithHighQuality : Starting downloading video with high quality (>720p).
func (dl *Downloader) DownloadWithHighQuality(ctx context.Context, outputFile string, v *youtube.Video, quality string) error {
	var videoFormat, audioFormat *youtube.Format

	if len(v.VideoFormats) == 0 {
		return fmt.Errorf("no format video for this video")
	}
	if len(quality) == 0 {
		videoFormat = &v.VideoFormats[0]
	} else {
		videoFormat = v.VideoFormats.FindByQuality(quality)
		if videoFormat == nil {
			return fmt.Errorf("no format video for '%s' found", quality)
		}
	}

	// Find suitable audio stream
	if strings.Contains(videoFormat.MimeType, "mp4") {
		for _, itag := range []int{258, 256, 140} {
			audioFormat = v.AudioFormats.FindByItag(itag)
			if audioFormat != nil {
				break
			}
		}
	} else if strings.Contains(videoFormat.MimeType, "webm") {
		for _, itag := range []int{251, 250, 249} {
			audioFormat = v.AudioFormats.FindByItag(itag)
			if audioFormat != nil {
				break
			}
		}
	} else {
		return fmt.Errorf("unhandled mime-type %s", videoFormat.MimeType)
	}

	if audioFormat == nil {
		return fmt.Errorf("no format audio for %s found", quality)
	}

	return dl.downloadVideoAudioStreamsAndJoin(ctx, outputFile, v, videoFormat, audioFormat)
}

func (dl *Downloader) downloadVideoAudioStreamsAndJoin(ctx context.Context, outputFile string, v *youtube.Video, videoFormat *youtube.Format, audioFormat *youtube.Format) error {
	dl.LogInfo("Video (Quality '%s' | FPS '%d' | Codec '%s')",
		videoFormat.QualityLabel, videoFormat.FPS, videoFormat.MimeType)
	dl.LogInfo("Audio (Channels '%d' | Codec '%s')",
		audioFormat.AudioChannels, audioFormat.MimeType)

	destFile, err := dl.getOutputFile(v, videoFormat, outputFile)
	if err != nil {
		return err
	}
	if fileExists(destFile) {
		return fmt.Errorf("⚠ SKIP: video %s - file %s exists", v.ID, destFile)
	}

	// Create temporary video file
	videoFile, err := ioutil.TempFile(os.TempDir(), "youtube_*.m4v")
	if err != nil {
		return err
	}
	defer os.Remove(videoFile.Name())

	// Create temporary audio file
	audioFile, err := ioutil.TempFile(os.TempDir(), "youtube_*.m4a")
	if err != nil {
		return err
	}
	defer os.Remove(audioFile.Name())

	dl.LogInfo("Downloading video file...")
	err = dl.videoDLWorker(ctx, videoFile, v, videoFormat)
	if err != nil {
		return err
	}

	dl.LogInfo("Downloading audio file...")
	err = dl.videoDLWorker(ctx, audioFile, v, audioFormat)
	if err != nil {
		return err
	}

	ffmpegVersionCmd := exec.Command("ffmpeg", "-y",
		"-i", videoFile.Name(),
		"-i", audioFile.Name(),
		"-c", "copy", // Just copy without re-encoding
		"-shortest", // Finish encoding when the shortest input stream ends
		destFile,
		"-loglevel", "warning",
	)
	ffmpegVersionCmd.Stderr = os.Stderr
	ffmpegVersionCmd.Stdout = os.Stdout
	dl.LogInfo("merging video and audio to %s", destFile)

	return ffmpegVersionCmd.Run()
}

func (dl *Downloader) videoDLWorker(ctx context.Context, out *os.File, video *youtube.Video, format *youtube.Format) error {
	resp, err := dl.GetStreamContext(ctx, video, format)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	prog := &progress{
		contentLength: float64(resp.ContentLength),
	}

	// create progress bar
	progress := mpb.New(mpb.WithWidth(64))
	bar := progress.AddBar(
		int64(prog.contentLength),

		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f"),
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)

	reader := bar.ProxyReader(resp.Body)
	mw := io.MultiWriter(out, prog)
	_, err = io.Copy(mw, reader)
	if err != nil {
		return err
	}

	progress.Wait()
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
