//go:build disabled

package cmd

import (
	"fmt"
	"mime"
	"path/filepath"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/spf13/cobra"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload <bucket> <local-file>",
	Short: "Upload a file to an R2 bucket",
	Long: `Upload a local file to a Cloudflare R2 bucket with a specified object key.

The --key flag specifies the remote object key (path in the bucket).
If not provided, it defaults to the filename portion of the local file path.

Examples:
  r2go2 upload my-bucket ./file.txt --key="remote/path/file.txt"
  r2go2 upload my-bucket ./image.jpg --key="images/photo.jpg"
  r2go2 upload my-bucket ./backup.zip --key="backups/$(date +%Y-%m-%d)/backup.zip"
  r2go2 upload my-bucket ./data.json --dry-run`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("exactly two arguments required: <bucket> <local-file>")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bucketName := args[0]
		localPath := args[1]
		objectKey, _ := cmd.Flags().GetString("key")

		// Default object key to filename if not provided
		if objectKey == "" {
			objectKey = filepath.Base(localPath)
		}

		// Create API client
		opts := &api.ClientOptions{
			AccountID: AccountID,
			APIToken:  APIToken,
		}
		client, err := api.NewClient(opts)
		if err != nil {
			printErrorAndExit(err, "Failed to create API client")
		}

		if Verbose {
			printInfo("Uploading file: %s", getRelativePath(localPath))
			printInfo("To bucket: %s", bucketName)
			printInfo("Object key: %s", objectKey)
		}

		// Detect content type
		contentType := mime.TypeByExtension(filepath.Ext(localPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		uploadRequest := &api.UploadRequest{
			Bucket:      bucketName,
			LocalPath:   localPath,
			ObjectKey:   objectKey,
			ContentType: contentType,
			Metadata:    make(map[string]string),
		}

		if DryRun {
			printSuccess("DRY RUN: Would upload '%s' to bucket '%s' as '%s'",
				getRelativePath(localPath), bucketName, objectKey)
			printInfo("Content type: %s", contentType)
			return
		}

		// Upload the file
		result, err := client.UploadObject(uploadRequest)
		if err != nil {
			printErrorAndExit(err, "Failed to upload file")
		}

		// Format file size for display
		sizeStr := utils.FormatBytes(result.Size)

		if JSONOutput {
			printSuccessJSON("File uploaded successfully", result)
		} else {
			printSuccess("File uploaded successfully")
			printInfo("Bucket: %s", bucketName)
			printInfo("Object key: %s", objectKey)
			printInfo("Size: %s", sizeStr)
			printInfo("ETag: %s", result.ETag)
			printInfo("Upload time: %s", result.UploadTime.Format("2006-01-02 15:04:05 UTC"))
		}
	},
}

func init() {
	uploadCmd.Flags().String("key", "", "Remote object key (defaults to filename)")
	uploadCmd.MarkFlagRequired("key") // Make this optional in future
}

