package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Manage Cloudflare Stream videos",
	Long: `Cloudflare Stream management — upload, manage, and deliver video.

Commands:
  upload    Upload a video (file or URL)
  list      List all videos
  get       Get video details
  delete    Delete a video
  token     Generate a signed playback token
  live      Manage live inputs

Cloudflare Stream is an account-scoped service for uploading, encoding,
and delivering video via HLS and DASH. Videos can require signed URLs
for playback and support custom metadata and watermarks.

Examples:
  cosmoflare stream upload video.mp4
  cosmoflare stream upload --url https://example.com/video.mp4
  cosmoflare stream list --json
  cosmoflare stream list --status ready
  cosmoflare stream get VIDEO_ID
  cosmoflare stream delete VIDEO_ID --force
  cosmoflare stream token VIDEO_ID --expires 1h
  cosmoflare stream live create my-stream --mode automatic
  cosmoflare stream live list
  cosmoflare stream live delete INPUT_ID --force`,
}

var streamLiveCmd = &cobra.Command{
	Use:   "live",
	Short: "Manage live inputs",
	Long: `Manage Cloudflare Stream live inputs.

Live inputs provide RTMPS, SRT, and WebRTC endpoints for live streaming.
Recordings can be stored automatically when recording mode is enabled.

Recording modes:
  off        No recording (default)
  automatic  Automatically record live streams

Commands:
  create    Create a new live input
  list      List all live inputs
  delete    Delete a live input

Examples:
  cosmoflare stream live create my-stream
  cosmoflare stream live create my-stream --mode automatic
  cosmoflare stream live list --json
  cosmoflare stream live delete INPUT_ID --force`,
}

var (
	streamURL              string
	streamMetadata         string
	streamWatermark        string
	streamRequireSignedURL bool
	streamForce            bool
	streamStatus           string
	streamExpires          string
	streamLiveMode         string
	streamLiveForce        bool
)

var streamUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload a video",
	Long: `Upload a video to Cloudflare Stream from a local file or URL.

Provide a local file path as an argument, or use --url for remote uploads.
Custom metadata can be attached as a JSON string.

Examples:
  cosmoflare stream upload video.mp4
  cosmoflare stream upload recording.mov --metadata '{"project":"docs"}'
  cosmoflare stream upload --url https://example.com/video.mp4
  cosmoflare stream upload --url https://example.com/vid.mp4 --require-signed-urls
  cosmoflare stream upload --url https://example.com/vid.mp4 --watermark WM_UID
  cosmoflare stream upload video.mp4 --json`,
	RunE: runStreamUpload,
}

var streamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all videos",
	Long: `List all videos in your Cloudflare Stream account.

Use --status to filter by processing state (ready, processing, error).

Examples:
  cosmoflare stream list
  cosmoflare stream list --json
  cosmoflare stream list --status ready
  cosmoflare stream list --status processing`,
	RunE: runStreamList,
}

var streamGetCmd = &cobra.Command{
	Use:   "get [video-id]",
	Short: "Get video details",
	Long: `Get details of a single video by its ID.

Displays playback URLs, status, dimensions, duration, and metadata.

Examples:
  cosmoflare stream get VIDEO_ID
  cosmoflare stream get VIDEO_ID --json`,
	RunE: runStreamGet,
}

var streamDeleteCmd = &cobra.Command{
	Use:   "delete [video-id]",
	Short: "Delete a video",
	Long: `Delete a video from Cloudflare Stream.

WARNING: This action is irreversible. The video and all its encodings
will be permanently removed.

Examples:
  cosmoflare stream delete VIDEO_ID
  cosmoflare stream delete VIDEO_ID --force
  cosmoflare stream delete VIDEO_ID --json`,
	RunE: runStreamDelete,
}

var streamTokenCmd = &cobra.Command{
	Use:   "token [video-id]",
	Short: "Generate a signed playback token",
	Long: `Generate a signed playback token for a video.

The token can be used to access videos that require signed URLs.
Use --expires to set the token lifetime (default: 1h).

Duration format: 1h, 30m, 24h, 7d (use Go duration or d suffix for days).

Examples:
  cosmoflare stream token VIDEO_ID
  cosmoflare stream token VIDEO_ID --expires 24h
  cosmoflare stream token VIDEO_ID --expires 7d
  cosmoflare stream token VIDEO_ID --json`,
	RunE: runStreamToken,
}

var streamLiveCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a live input",
	Long: `Create a new Cloudflare Stream live input.

A live input provides RTMPS, SRT, and WebRTC endpoints for live streaming.
Use --mode automatic to enable automatic recording of live streams.

Recording modes:
  off        No recording (default)
  automatic  Automatically record live streams

Examples:
  cosmoflare stream live create my-stream
  cosmoflare stream live create my-stream --mode automatic
  cosmoflare stream live create my-stream --json`,
	RunE: runStreamLiveCreate,
}

var streamLiveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all live inputs",
	Long: `List all live inputs in your Cloudflare Stream account.

Examples:
  cosmoflare stream live list
  cosmoflare stream live list --json`,
	RunE: runStreamLiveList,
}

var streamLiveDeleteCmd = &cobra.Command{
	Use:   "delete [input-id]",
	Short: "Delete a live input",
	Long: `Delete a live input from Cloudflare Stream.

WARNING: This action is irreversible. The live input and its endpoints
will be permanently removed.

Examples:
  cosmoflare stream live delete INPUT_ID
  cosmoflare stream live delete INPUT_ID --force`,
	RunE: runStreamLiveDelete,
}

func init() {
	rootCmd.AddCommand(streamCmd)

	streamCmd.AddCommand(streamUploadCmd)
	streamCmd.AddCommand(streamListCmd)
	streamCmd.AddCommand(streamGetCmd)
	streamCmd.AddCommand(streamDeleteCmd)
	streamCmd.AddCommand(streamTokenCmd)
	streamCmd.AddCommand(streamLiveCmd)

	streamLiveCmd.AddCommand(streamLiveCreateCmd)
	streamLiveCmd.AddCommand(streamLiveListCmd)
	streamLiveCmd.AddCommand(streamLiveDeleteCmd)

	// Upload flags
	streamUploadCmd.Flags().StringVar(&streamURL, "url", "", "Upload video from URL instead of local file")
	streamUploadCmd.Flags().StringVar(&streamMetadata, "metadata", "", "Custom metadata as JSON (e.g., '{\"key\":\"value\"}')")
	streamUploadCmd.Flags().StringVar(&streamWatermark, "watermark", "", "Watermark UID to apply to the video")
	streamUploadCmd.Flags().BoolVar(&streamRequireSignedURL, "require-signed-urls", false, "Require signed URLs for video playback")

	// List flags
	streamListCmd.Flags().StringVar(&streamStatus, "status", "", "Filter by status (ready, processing, error)")

	// Delete flags
	streamDeleteCmd.Flags().BoolVar(&streamForce, "force", false, "Skip confirmation prompt")

	// Token flags
	streamTokenCmd.Flags().StringVar(&streamExpires, "expires", "1h", "Token expiration duration (e.g., 1h, 24h, 7d)")

	// Live create flags
	streamLiveCreateCmd.Flags().StringVar(&streamLiveMode, "mode", "", "Recording mode (off, automatic)")

	// Live delete flags
	streamLiveDeleteCmd.Flags().BoolVar(&streamLiveForce, "force", false, "Skip confirmation prompt")
}

func getStreamService() (*cosmoflare.StreamService, error) {
	return cosmoflare.NewStreamServiceFromCreds(AccountID, APIToken)
}

// parseExpiresDuration parses a duration string, supporting "d" suffix for days.
func parseExpiresDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func runStreamUpload(cmd *cobra.Command, args []string) error {
	// URL upload mode
	if streamURL != "" {
		return runStreamUploadByURL(cmd)
	}

	// File upload mode — requires a file argument
	if len(args) == 0 {
		return fmt.Errorf("file path is required (or use --url for URL upload)")
	}
	filePath := args[0]

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would upload video", map[string]interface{}{
				"file": filePath,
			})
		}
		printInfo("DRY RUN: Would upload video from '%s'", filePath)
		return nil
	}

	// Verify file exists before creating service
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	var metadata map[string]interface{}
	if streamMetadata != "" {
		if err := json.Unmarshal([]byte(streamMetadata), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	video, err := svc.UploadFile(context.Background(), filePath, metadata)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to upload video: %v", err))
		}
		return fmt.Errorf("failed to upload video: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Video uploaded successfully", video)
	}

	printSuccess("Video uploaded successfully!")
	printInfo("UID: %s", video.UID)
	printInfo("Status: %s", video.Status.State)
	if video.Duration > 0 {
		printInfo("Duration: %.1fs", video.Duration)
	}
	return nil
}

func runStreamUploadByURL(cmd *cobra.Command) error {
	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would upload video from URL", map[string]interface{}{
				"url": streamURL,
			})
		}
		printInfo("DRY RUN: Would upload video from URL '%s'", streamURL)
		return nil
	}

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	var opts []cosmoflare.StreamUploadOption
	if streamRequireSignedURL {
		opts = append(opts, cosmoflare.WithStreamRequireSignedURLs(true))
	}
	if streamMetadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(streamMetadata), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
		opts = append(opts, cosmoflare.WithStreamMetadata(metadata))
	}
	if streamWatermark != "" {
		opts = append(opts, cosmoflare.WithStreamWatermark(streamWatermark))
	}

	video, err := svc.UploadByURL(context.Background(), streamURL, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to upload video from URL: %v", err))
		}
		return fmt.Errorf("failed to upload video from URL: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Video uploaded from URL successfully", video)
	}

	printSuccess("Video uploaded from URL successfully!")
	printInfo("UID: %s", video.UID)
	printInfo("Status: %s", video.Status.State)
	return nil
}

func runStreamList(cmd *cobra.Command, args []string) error {
	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	videos, err := svc.ListVideos(context.Background(), streamStatus)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list videos: %v", err))
		}
		return fmt.Errorf("failed to list videos: %w", err)
	}

	if JSONOutput {
		return printJSON(videos)
	}

	if len(videos) == 0 {
		printInfo("No videos found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UID\tSTATUS\tDURATION\tSIZE\tREADY\tSIGNED")
	for _, v := range videos {
		readyStr := "no"
		if v.ReadyToStream {
			readyStr = "yes"
		}
		signedStr := "no"
		if v.RequireSignedURLs {
			signedStr = "yes"
		}
		sizeStr := formatBytes(int64(v.Size))
		fmt.Fprintf(w, "%s\t%s\t%.1fs\t%s\t%s\t%s\n",
			v.UID,
			v.Status.State,
			v.Duration,
			sizeStr,
			readyStr,
			signedStr,
		)
	}
	w.Flush()

	printInfo("Total: %d video(s)", len(videos))
	return nil
}

func runStreamGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("video ID is required")
	}
	videoID := args[0]

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	video, err := svc.GetVideo(context.Background(), videoID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get video: %v", err))
		}
		return fmt.Errorf("failed to get video: %w", err)
	}

	if JSONOutput {
		return printJSON(video)
	}

	fmt.Printf("UID:               %s\n", video.UID)
	fmt.Printf("Status:            %s\n", video.Status.State)
	if video.Status.PctComplete != "" {
		fmt.Printf("Progress:          %s%%\n", video.Status.PctComplete)
	}
	fmt.Printf("Ready to Stream:   %v\n", video.ReadyToStream)
	fmt.Printf("Duration:          %.1fs\n", video.Duration)
	fmt.Printf("Size:              %s\n", formatBytes(int64(video.Size)))
	if video.Input.Width > 0 && video.Input.Height > 0 {
		fmt.Printf("Resolution:        %dx%d\n", video.Input.Width, video.Input.Height)
	}
	fmt.Printf("Require Signed:    %v\n", video.RequireSignedURLs)
	if video.Playback.HLS != "" {
		fmt.Printf("HLS URL:           %s\n", video.Playback.HLS)
	}
	if video.Playback.Dash != "" {
		fmt.Printf("DASH URL:          %s\n", video.Playback.Dash)
	}
	if video.Preview != "" {
		fmt.Printf("Preview:           %s\n", video.Preview)
	}
	if video.Thumbnail != "" {
		fmt.Printf("Thumbnail:         %s\n", video.Thumbnail)
	}
	if video.Created != nil {
		fmt.Printf("Created:           %s\n", video.Created.Format("2006-01-02 15:04:05"))
	}
	if len(video.Meta) > 0 {
		metaJSON, _ := json.MarshalIndent(video.Meta, "                   ", "  ")
		fmt.Printf("Metadata:          %s\n", string(metaJSON))
	}
	return nil
}

func runStreamDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("video ID is required")
	}
	videoID := args[0]

	if !streamForce && !DryRun {
		fmt.Printf("Are you sure you want to delete video '%s'? [y/N]: ", videoID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Video deletion cancelled")
			return nil
		}
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete video", map[string]string{
				"video_id": videoID,
			})
		}
		printInfo("DRY RUN: Would delete video '%s'", videoID)
		return nil
	}

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	if err := svc.DeleteVideo(context.Background(), videoID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete video: %v", err))
		}
		return fmt.Errorf("failed to delete video: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Video deleted successfully", map[string]string{
			"video_id": videoID,
		})
	}
	printSuccess("Video '%s' deleted successfully!", videoID)
	return nil
}

func runStreamToken(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("video ID is required")
	}
	videoID := args[0]

	dur, err := parseExpiresDuration(streamExpires)
	if err != nil {
		return fmt.Errorf("invalid --expires value: %w", err)
	}
	expiresSeconds := int(dur.Seconds())

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would generate signed token", map[string]interface{}{
				"video_id": videoID,
				"expires":  streamExpires,
			})
		}
		printInfo("DRY RUN: Would generate signed token for video '%s' (expires: %s)", videoID, streamExpires)
		return nil
	}

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	token, err := svc.CreateSignedToken(context.Background(), videoID, expiresSeconds)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create signed token: %v", err))
		}
		return fmt.Errorf("failed to create signed token: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Signed token generated", map[string]string{
			"video_id": videoID,
			"token":    token,
			"expires":  streamExpires,
		})
	}

	printSuccess("Signed playback token generated!")
	printInfo("Video: %s", videoID)
	printInfo("Expires: %s", streamExpires)
	fmt.Println()
	fmt.Println(token)
	return nil
}

func runStreamLiveCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("live input name is required")
	}
	name := args[0]

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create live input", map[string]interface{}{
				"name": name,
				"mode": streamLiveMode,
			})
		}
		printInfo("DRY RUN: Would create live input '%s'", name)
		if streamLiveMode != "" {
			printInfo("Recording mode: %s", streamLiveMode)
		}
		return nil
	}

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	input, err := svc.CreateLiveInput(context.Background(), name, streamLiveMode)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create live input: %v", err))
		}
		return fmt.Errorf("failed to create live input: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Live input created successfully", input)
	}

	printSuccess("Live input '%s' created successfully!", name)
	printInfo("UID: %s", input.UID)
	if input.RTMPS.URL != "" {
		printInfo("RTMPS URL: %s", input.RTMPS.URL)
		printInfo("RTMPS Stream Key: %s", input.RTMPS.StreamKey)
	}
	if input.SRT.URL != "" {
		printInfo("SRT URL: %s", input.SRT.URL)
	}
	if input.WebRTC.URL != "" {
		printInfo("WebRTC URL: %s", input.WebRTC.URL)
	}
	return nil
}

func runStreamLiveList(cmd *cobra.Command, args []string) error {
	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	inputs, err := svc.ListLiveInputs(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list live inputs: %v", err))
		}
		return fmt.Errorf("failed to list live inputs: %w", err)
	}

	if JSONOutput {
		return printJSON(inputs)
	}

	if len(inputs) == 0 {
		printInfo("No live inputs found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UID\tNAME\tSTATUS\tRECORDING\tCREATED")
	for _, inp := range inputs {
		name := ""
		if inp.Meta != nil {
			if n, ok := inp.Meta["name"].(string); ok {
				name = n
			}
		}
		created := ""
		if inp.Created != nil {
			created = inp.Created.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			inp.UID,
			name,
			inp.Status,
			inp.Recording.Mode,
			created,
		)
	}
	w.Flush()

	printInfo("Total: %d live input(s)", len(inputs))
	return nil
}

func runStreamLiveDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("live input ID is required")
	}
	inputID := args[0]

	if !streamLiveForce && !DryRun {
		fmt.Printf("Are you sure you want to delete live input '%s'? [y/N]: ", inputID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Live input deletion cancelled")
			return nil
		}
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete live input", map[string]string{
				"input_id": inputID,
			})
		}
		printInfo("DRY RUN: Would delete live input '%s'", inputID)
		return nil
	}

	svc, err := getStreamService()
	if err != nil {
		return fmt.Errorf("failed to create Stream service: %w", err)
	}

	if err := svc.DeleteLiveInput(context.Background(), inputID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete live input: %v", err))
		}
		return fmt.Errorf("failed to delete live input: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Live input deleted successfully", map[string]string{
			"input_id": inputID,
		})
	}
	printSuccess("Live input '%s' deleted successfully!", inputID)
	return nil
}

