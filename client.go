package youtube

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Client offers methods to download video metadata and video streams.
type Client struct {
	// LogLevel
	logLevel string

	// HTTPClient can be used to set a custom HTTP client.
	// If not set, http.DefaultClient will be used
	HTTPClient *http.Client
}

var logLevelMapping = map[string]int{
	"off":       0,
	"emergency": 1,
	"critical":  2,
	"error":     3,
	"warning":   4,
	"info":      5,
	"debug":     6,
	"trace":     7,
}

func (c *Client) LogLevel() string {
	if len(c.logLevel) == 0 {
		c.SetLogLevel("off")
	}
	return c.logLevel
}

func (c *Client) SetLogLevel(logLevel string) *Client {
	if _, ok := logLevelMapping[logLevel]; !ok {
		panic("invalid log level")
	}
	c.logLevel = logLevel
	return c
}

func (c *Client) ShouldLog(logLevel string) bool {
	if v, ok := logLevelMapping[logLevel]; !ok {
		panic("invalid log level")
	} else {
		return v <= logLevelMapping[c.LogLevel()]
	}
}

// GetVideo fetches video metadata
func (c *Client) GetVideo(url string) (*Video, error) {
	return c.GetVideoContext(context.Background(), url)
}

// GetVideoContext fetches video metadata with a context
func (c *Client) GetVideoContext(ctx context.Context, url string) (*Video, error) {
	id, err := extractVideoID(url)
	if err != nil {
		return nil, fmt.Errorf("extractVideoID failed: %w", err)
	}

	// Circumvent age restriction to pretend access through googleapis.com
	eurl := "https://youtube.googleapis.com/v/" + id
	body, err := c.httpGetBodyBytes(ctx, "https://youtube.com/get_video_info?video_id="+id+"&eurl="+eurl)
	if err != nil {
		return nil, err
	}

	v := &Video{
		ID: id,
	}

	return v, v.parseVideoInfo(string(body))
}

// GetStream returns the HTTP response for a specific format
func (c *Client) GetStream(video *Video, format *Format) (*http.Response, error) {
	return c.GetStreamContext(context.Background(), video, format)
}

// GetStreamContext returns the HTTP response for a specific format with a context
func (c *Client) GetStreamContext(ctx context.Context, video *Video, format *Format) (*http.Response, error) {
	url, err := c.GetStreamURLContext(ctx, video, format)
	if err != nil {
		return nil, err
	}

	return c.httpGet(ctx, url)
}

// GetStreamURL returns the url for a specific format
func (c *Client) GetStreamURL(video *Video, format *Format) (string, error) {
	return c.GetStreamURLContext(context.Background(), video, format)
}

// GetStreamURL returns the url for a specific format with a context
func (c *Client) GetStreamURLContext(ctx context.Context, video *Video, format *Format) (string, error) {
	if format.URL != "" {
		return format.URL, nil
	}

	cipher := format.Cipher
	if cipher == "" {
		return "", ErrCipherNotFound
	}

	return c.decipherURL(ctx, video.ID, cipher)
}

// httpGet does a HTTP GET request, checks the response to be a 200 OK and returns it
func (c *Client) httpGet(ctx context.Context, url string) (resp *http.Response, err error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	c.LogTrace("GET %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, ErrUnexpectedStatusCode(resp.StatusCode)
	}

	return
}

// httpGetBodyBytes reads the whole HTTP body and returns it
func (c *Client) httpGetBodyBytes(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (c *Client) LogInfo(format string, v ...interface{}) {
	if c.ShouldLog("info") {
		log.Printf(format, v...)
	}
}

func (c *Client) LogDebug(format string, v ...interface{}) {
	if c.ShouldLog("debug") {
		log.Printf(format, v...)
	}
}

func (c *Client) LogTrace(format string, v ...interface{}) {
	if c.ShouldLog("trace") {
		log.Printf(format, v...)
	}
}
