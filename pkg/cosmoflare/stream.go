package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// CFStreamVideo represents a Cloudflare Stream video.
type CFStreamVideo struct {
	UID               string                 `json:"uid"`
	Created           *time.Time             `json:"created,omitempty"`
	Modified          *time.Time             `json:"modified,omitempty"`
	Duration          float64                `json:"duration,omitempty"`
	Size              int                    `json:"size,omitempty"`
	ReadyToStream     bool                   `json:"readyToStream"`
	RequireSignedURLs bool                   `json:"requireSignedURLs"`
	Status            CFStreamVideoStatus    `json:"status"`
	Input             CFStreamVideoInput     `json:"input,omitempty"`
	Playback          CFStreamVideoPlayback  `json:"playback,omitempty"`
	Preview           string                 `json:"preview,omitempty"`
	Thumbnail         string                 `json:"thumbnail,omitempty"`
	Meta              map[string]interface{} `json:"meta,omitempty"`
	Creator           string                 `json:"creator,omitempty"`
	LiveInput         string                 `json:"liveInput,omitempty"`
	Uploaded          *time.Time             `json:"uploaded,omitempty"`
	Watermark         CFStreamWatermark      `json:"watermark,omitempty"`
}

// CFStreamVideoStatus represents the processing status of a Stream video.
type CFStreamVideoStatus struct {
	State           string `json:"state,omitempty"`
	PctComplete     string `json:"pctComplete,omitempty"`
	ErrorReasonCode string `json:"errorReasonCode,omitempty"`
	ErrorReasonText string `json:"errorReasonText,omitempty"`
}

// CFStreamVideoInput represents the input dimensions of a video.
type CFStreamVideoInput struct {
	Height int `json:"height,omitempty"`
	Width  int `json:"width,omitempty"`
}

// CFStreamVideoPlayback holds HLS and DASH playback URLs.
type CFStreamVideoPlayback struct {
	HLS  string `json:"hls,omitempty"`
	Dash string `json:"dash,omitempty"`
}

// CFStreamWatermark represents a watermark on a Stream video.
type CFStreamWatermark struct {
	UID string `json:"uid,omitempty"`
}

// CFStreamLiveInput represents a Cloudflare Stream live input.
type CFStreamLiveInput struct {
	UID              string                    `json:"uid"`
	Created          *time.Time                `json:"created,omitempty"`
	Modified         *time.Time                `json:"modified,omitempty"`
	Meta             map[string]interface{}    `json:"meta,omitempty"`
	Status           string                    `json:"status,omitempty"`
	Recording        CFStreamLiveRecording     `json:"recording,omitempty"`
	RTMPS            CFStreamLiveRTMPS         `json:"rtmps,omitempty"`
	RTMPSPLAYBACK    CFStreamLiveRTMPS         `json:"rtmpsPlayback,omitempty"`
	SRT              CFStreamLiveSRT           `json:"srt,omitempty"`
	SRTPlayback      CFStreamLiveSRT           `json:"srtPlayback,omitempty"`
	WebRTC           CFStreamLiveWebRTC        `json:"webRTC,omitempty"`
	WebRTCPlayback   CFStreamLiveWebRTC        `json:"webRTCPlayback,omitempty"`
	DeleteRecording  bool                      `json:"deleteRecordingAfterDays,omitempty"`
}

// CFStreamLiveRecording holds recording settings for a live input.
type CFStreamLiveRecording struct {
	Mode              string `json:"mode,omitempty"`
	RequireSignedURLs bool   `json:"requireSignedURLs,omitempty"`
	AllowedOrigins    []string `json:"allowedOrigins,omitempty"`
	TimeoutSeconds    int    `json:"timeoutSeconds,omitempty"`
}

// CFStreamLiveRTMPS holds RTMPS connection details.
type CFStreamLiveRTMPS struct {
	URL       string `json:"url,omitempty"`
	StreamKey string `json:"streamKey,omitempty"`
}

// CFStreamLiveSRT holds SRT connection details.
type CFStreamLiveSRT struct {
	URL        string `json:"url,omitempty"`
	StreamID   string `json:"streamId,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// CFStreamLiveWebRTC holds WebRTC connection details.
type CFStreamLiveWebRTC struct {
	URL string `json:"url,omitempty"`
}

// StreamService implements Cloudflare Stream video operations.
type StreamService struct {
	cf        *cloudflare.API
	accountID string
}

// NewStreamService creates a new Stream service client.
func NewStreamService(api *cloudflare.API, accountID string) (*StreamService, error) {
	if api == nil {
		return nil, validationError("NewStreamService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewStreamService", "account ID is required")
	}
	return &StreamService{cf: api, accountID: accountID}, nil
}

// NewStreamServiceFromCreds creates a StreamService from account ID and API token.
func NewStreamServiceFromCreds(accountID, apiToken string) (*StreamService, error) {
	if accountID == "" {
		return nil, validationError("NewStreamService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewStreamService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewStreamService", "failed to create Cloudflare API client", err)
	}
	return &StreamService{cf: cf, accountID: accountID}, nil
}

// UploadFile uploads a video from a local file path.
func (s *StreamService) UploadFile(ctx context.Context, filePath string, metadata map[string]interface{}) (*CFStreamVideo, error) {
	if filePath == "" {
		return nil, validationError("StreamService.UploadFile", "file path is required")
	}

	params := cloudflare.StreamUploadFileParameters{
		AccountID: s.accountID,
		FilePath:  filePath,
	}

	video, err := s.cf.StreamUploadVideoFile(ctx, params)
	if err != nil {
		return nil, newError("StreamService.UploadFile", "failed to upload video file", err)
	}

	return toCFStreamVideo(video), nil
}

// UploadByURL uploads a video by downloading from a remote URL.
func (s *StreamService) UploadByURL(ctx context.Context, url string, opts ...StreamUploadOption) (*CFStreamVideo, error) {
	if url == "" {
		return nil, validationError("StreamService.UploadByURL", "URL is required")
	}

	cfg := &streamUploadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	params := cloudflare.StreamUploadFromURLParameters{
		AccountID:         s.accountID,
		URL:               url,
		RequireSignedURLs: cfg.requireSignedURLs,
		Meta:              cfg.metadata,
	}
	if cfg.watermarkUID != "" {
		params.Watermark = cloudflare.UploadVideoURLWatermark{UID: cfg.watermarkUID}
	}

	video, err := s.cf.StreamUploadFromURL(ctx, params)
	if err != nil {
		return nil, newError("StreamService.UploadByURL", "failed to upload video from URL", err)
	}

	return toCFStreamVideo(video), nil
}

// ListVideos returns all videos, optionally filtered by status.
func (s *StreamService) ListVideos(ctx context.Context, status string) ([]*CFStreamVideo, error) {
	params := cloudflare.StreamListParameters{
		AccountID: s.accountID,
	}
	if status != "" {
		params.Status = status
	}

	videos, err := s.cf.StreamListVideos(ctx, params)
	if err != nil {
		return nil, newError("StreamService.ListVideos", "failed to list videos", err)
	}

	result := make([]*CFStreamVideo, 0, len(videos))
	for _, v := range videos {
		result = append(result, toCFStreamVideo(v))
	}
	return result, nil
}

// GetVideo retrieves details for a single video.
func (s *StreamService) GetVideo(ctx context.Context, videoID string) (*CFStreamVideo, error) {
	if videoID == "" {
		return nil, validationError("StreamService.GetVideo", "video ID is required")
	}

	video, err := s.cf.StreamGetVideo(ctx, cloudflare.StreamParameters{
		AccountID: s.accountID,
		VideoID:   videoID,
	})
	if err != nil {
		return nil, notFound("StreamService.GetVideo", "", videoID, err)
	}

	return toCFStreamVideo(video), nil
}

// DeleteVideo deletes a video by ID.
func (s *StreamService) DeleteVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return validationError("StreamService.DeleteVideo", "video ID is required")
	}

	err := s.cf.StreamDeleteVideo(ctx, cloudflare.StreamParameters{
		AccountID: s.accountID,
		VideoID:   videoID,
	})
	if err != nil {
		return newError("StreamService.DeleteVideo", fmt.Sprintf("failed to delete video %q", videoID), err)
	}
	return nil
}

// CreateSignedToken generates a signed playback token for a video.
func (s *StreamService) CreateSignedToken(ctx context.Context, videoID string, expiresInSeconds int) (string, error) {
	if videoID == "" {
		return "", validationError("StreamService.CreateSignedToken", "video ID is required")
	}

	params := cloudflare.StreamSignedURLParameters{
		AccountID: s.accountID,
		VideoID:   videoID,
	}
	if expiresInSeconds > 0 {
		params.EXP = int(time.Now().Add(time.Duration(expiresInSeconds) * time.Second).Unix())
	}

	token, err := s.cf.StreamCreateSignedURL(ctx, params)
	if err != nil {
		return "", newError("StreamService.CreateSignedToken", "failed to create signed token", err)
	}
	return token, nil
}

// --- Live Input methods (direct API — not in cloudflare-go SDK) ---

// liveInputResponse represents the API response for a live input.
type liveInputResponse struct {
	Success  bool              `json:"success"`
	Errors   []interface{}     `json:"errors"`
	Messages []interface{}     `json:"messages"`
	Result   CFStreamLiveInput `json:"result"`
}

// liveInputListResponse represents the API response for listing live inputs.
type liveInputListResponse struct {
	Success  bool                `json:"success"`
	Errors   []interface{}       `json:"errors"`
	Messages []interface{}       `json:"messages"`
	Result   []CFStreamLiveInput `json:"result"`
}

// CreateLiveInput creates a new live input.
func (s *StreamService) CreateLiveInput(ctx context.Context, name string, mode string) (*CFStreamLiveInput, error) {
	if name == "" {
		return nil, validationError("StreamService.CreateLiveInput", "live input name is required")
	}

	body := map[string]interface{}{
		"meta": map[string]interface{}{
			"name": name,
		},
	}
	if mode != "" {
		body["recording"] = map[string]interface{}{
			"mode": mode,
		}
	}

	uri := fmt.Sprintf("/accounts/%s/stream/live_inputs", s.accountID)
	raw, err := s.cf.Raw(ctx, http.MethodPost, uri, body, nil)
	if err != nil {
		return nil, newError("StreamService.CreateLiveInput", "failed to create live input", err)
	}

	var input CFStreamLiveInput
	if err := json.Unmarshal(raw.Result, &input); err != nil {
		return nil, newError("StreamService.CreateLiveInput", "failed to parse response", err)
	}

	return &input, nil
}

// ListLiveInputs returns all live inputs.
func (s *StreamService) ListLiveInputs(ctx context.Context) ([]*CFStreamLiveInput, error) {
	uri := fmt.Sprintf("/accounts/%s/stream/live_inputs", s.accountID)
	raw, err := s.cf.Raw(ctx, http.MethodGet, uri, nil, nil)
	if err != nil {
		return nil, newError("StreamService.ListLiveInputs", "failed to list live inputs", err)
	}

	// The API may return the list directly or wrapped in a "live_inputs" key
	var items []CFStreamLiveInput
	if err := json.Unmarshal(raw.Result, &items); err != nil {
		// Try wrapped response
		var wrapped struct {
			LiveInputs []CFStreamLiveInput `json:"live_inputs"`
		}
		if err2 := json.Unmarshal(raw.Result, &wrapped); err2 != nil {
			return nil, newError("StreamService.ListLiveInputs", "failed to parse response", err)
		}
		items = wrapped.LiveInputs
	}

	result := make([]*CFStreamLiveInput, 0, len(items))
	for i := range items {
		result = append(result, &items[i])
	}
	return result, nil
}

// DeleteLiveInput deletes a live input by ID.
func (s *StreamService) DeleteLiveInput(ctx context.Context, inputID string) error {
	if inputID == "" {
		return validationError("StreamService.DeleteLiveInput", "live input ID is required")
	}

	uri := fmt.Sprintf("/accounts/%s/stream/live_inputs/%s", s.accountID, inputID)
	_, err := s.cf.Raw(ctx, http.MethodDelete, uri, nil, nil)
	if err != nil {
		return newError("StreamService.DeleteLiveInput", fmt.Sprintf("failed to delete live input %q", inputID), err)
	}
	return nil
}

// --- Stream upload options ---

// StreamUploadOption is a functional option for stream upload operations.
type StreamUploadOption func(*streamUploadConfig)

type streamUploadConfig struct {
	requireSignedURLs bool
	metadata          map[string]interface{}
	watermarkUID      string
}

// WithStreamRequireSignedURLs sets whether the video requires signed URLs for playback.
func WithStreamRequireSignedURLs(v bool) StreamUploadOption {
	return func(c *streamUploadConfig) { c.requireSignedURLs = v }
}

// WithStreamMetadata attaches custom metadata to an uploaded video.
func WithStreamMetadata(m map[string]interface{}) StreamUploadOption {
	return func(c *streamUploadConfig) { c.metadata = m }
}

// WithStreamWatermark sets the watermark UID for the uploaded video.
func WithStreamWatermark(uid string) StreamUploadOption {
	return func(c *streamUploadConfig) { c.watermarkUID = uid }
}

// --- Conversion helpers ---

func toCFStreamVideo(v cloudflare.StreamVideo) *CFStreamVideo {
	return &CFStreamVideo{
		UID:               v.UID,
		Created:           v.Created,
		Modified:          v.Modified,
		Duration:          v.Duration,
		Size:              v.Size,
		ReadyToStream:     v.ReadyToStream,
		RequireSignedURLs: v.RequireSignedURLs,
		Status: CFStreamVideoStatus{
			State:           v.Status.State,
			PctComplete:     v.Status.PctComplete,
			ErrorReasonCode: v.Status.ErrorReasonCode,
			ErrorReasonText: v.Status.ErrorReasonText,
		},
		Input: CFStreamVideoInput{
			Height: v.Input.Height,
			Width:  v.Input.Width,
		},
		Playback: CFStreamVideoPlayback{
			HLS:  v.Playback.HLS,
			Dash: v.Playback.Dash,
		},
		Preview:   v.Preview,
		Thumbnail: v.Thumbnail,
		Meta:      v.Meta,
		Creator:   v.Creator,
		LiveInput: v.LiveInput,
		Uploaded:  v.Uploaded,
		Watermark: CFStreamWatermark{
			UID: v.Watermark.UID,
		},
	}
}
