package youtube

import (
	"strconv"
	"strings"
)

type FormatList []Format

// FindByQuality returns the format by quality or quality label, stream can be "", "audio", "video", this is used to
// match with mime-type
func (list FormatList) FindByQuality(quality string) *Format {
	for i := range list {
		if list[i].Quality == quality || list[i].QualityLabel == quality {
			return &list[i]
		}
	}
	return nil
}

func (list FormatList) FindByItag(itagNo int) *Format {
	for i := range list {
		if list[i].ItagNo == itagNo {
			return &list[i]
		}
	}
	return nil
}

// FilterCodec returns a new FormatList filtered by codec
// All codecs must be matched
func (list FormatList) FilterCodec(codec []string) FormatList {
	var formats FormatList
FormatLoop:
	for _, f := range list {
		for _, c := range codec {
			if !strings.Contains(f.MimeType, c) {
				continue FormatLoop
			}
		}
		formats = append(formats, f)
	}
	return formats
}

// FilterQuality returns a new FormatList filtered by quality
// For any matched quality
func (list FormatList) FilterQuality(quality []string) FormatList {
	var formats FormatList
FormatLoop:
	for _, f := range list {
		for _, q := range quality {
			itag, _ := strconv.Atoi(q)
			if itag == f.ItagNo {
				formats = append(formats, f)
				continue FormatLoop
			}
			if strings.Contains(f.Quality, q) || strings.Contains(f.QualityLabel, q) {
				formats = append(formats, f)
				continue FormatLoop
			}
		}
	}
	return formats
}

// FindByType returns mime type of video which only audio or video
func (list FormatList) FindByType(t string) []Format {
	var f []Format
	for i := range list {
		if strings.Contains(list[i].MimeType, t) {
			f = append(f, list[i])
		}
	}
	return f
}
