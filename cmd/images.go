package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage Cloudflare Images",
	Long: `Cloudflare Images management — upload, list, get, delete images
and manage delivery variants for resizing and transformation.

Commands:
  upload    Upload an image (file or URL)
  list      List all images
  get       Get image details
  delete    Delete an image
  variants  Manage image delivery variants

Cloudflare Images is an account-scoped service for storing, resizing,
and serving optimized images. Images are delivered through variants
that define resize and transformation parameters.

Examples:
  cosmoflare images upload photo.jpg
  cosmoflare images upload --url https://example.com/photo.jpg
  cosmoflare images list --json
  cosmoflare images get IMG_ID
  cosmoflare images delete IMG_ID --force
  cosmoflare images variants list
  cosmoflare images variants create thumbnail --fit=cover --width=150 --height=150`,
}

var imagesVariantsCmd = &cobra.Command{
	Use:   "variants",
	Short: "Manage image delivery variants",
	Long: `Manage Cloudflare Images delivery variants.

Variants define how images are resized and transformed when delivered.
Each variant specifies fit mode, dimensions, and metadata handling.

Fit modes:
  scale-down  Shrink to fit within dimensions (default, never upscale)
  contain     Fit within dimensions, preserving aspect ratio
  cover       Fill dimensions, cropping if needed
  crop        Crop to exact dimensions
  pad         Fit within dimensions, padding remaining space

Commands:
  list      List all variants
  create    Create a new variant
  delete    Delete a variant

Examples:
  cosmoflare images variants list --json
  cosmoflare images variants create hero --fit=cover --width=1200 --height=630
  cosmoflare images variants create thumb --fit=cover --width=150 --height=150
  cosmoflare images variants delete old-variant --force`,
}

var (
	imagesURL              string
	imagesMetadata         string
	imagesRequireSignedURL bool
	imagesForce            bool
	variantFit             string
	variantWidth           int
	variantHeight          int
	variantMetadataMode    string
	variantNeverSignedURLs bool
	variantForce           bool
)

var imagesUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload an image",
	Long: `Upload an image to Cloudflare Images from a local file or URL.

Provide a local file path as an argument, or use --url for remote uploads.
Custom metadata can be attached as a JSON string.

Examples:
  cosmoflare images upload photo.jpg
  cosmoflare images upload banner.png --metadata '{"project":"website"}'
  cosmoflare images upload --url https://example.com/photo.jpg
  cosmoflare images upload --url https://example.com/img.png --require-signed-urls
  cosmoflare images upload photo.jpg --json`,
	RunE: runImagesUpload,
}

var imagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all images",
	Long: `List all images in your Cloudflare Images account.

Examples:
  cosmoflare images list
  cosmoflare images list --json`,
	RunE: runImagesList,
}

var imagesGetCmd = &cobra.Command{
	Use:   "get [image-id]",
	Short: "Get image details",
	Long: `Get details of a single image by its ID.

Examples:
  cosmoflare images get IMG_ID
  cosmoflare images get IMG_ID --json`,
	RunE: runImagesGet,
}

var imagesDeleteCmd = &cobra.Command{
	Use:   "delete [image-id]",
	Short: "Delete an image",
	Long: `Delete an image from Cloudflare Images.

WARNING: This action is irreversible.

Examples:
  cosmoflare images delete IMG_ID
  cosmoflare images delete IMG_ID --force
  cosmoflare images delete IMG_ID --json`,
	RunE: runImagesDelete,
}

var imagesVariantsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all delivery variants",
	Long: `List all image delivery variants configured for your account.

Examples:
  cosmoflare images variants list
  cosmoflare images variants list --json`,
	RunE: runImagesVariantsList,
}

var imagesVariantsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a delivery variant",
	Long: `Create a new image delivery variant with resize/transform options.

Fit modes:
  scale-down  Shrink to fit within dimensions (default)
  contain     Fit within dimensions, preserving aspect ratio
  cover       Fill dimensions, cropping if needed
  crop        Crop to exact dimensions
  pad         Fit within dimensions, padding remaining space

Metadata modes:
  none        Strip all metadata (default)
  keep        Preserve all metadata
  copyright   Keep only copyright-related metadata

Examples:
  cosmoflare images variants create hero --fit=cover --width=1200 --height=630
  cosmoflare images variants create thumb --fit=cover --width=150 --height=150
  cosmoflare images variants create avatar --fit=crop --width=100 --height=100 --metadata-mode=none
  cosmoflare images variants create public --fit=scale-down --width=1920 --height=1080 --never-require-signed-urls
  cosmoflare images variants create hero --json`,
	RunE: runImagesVariantsCreate,
}

var imagesVariantsDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a delivery variant",
	Long: `Delete an image delivery variant.

WARNING: Deleting a variant purges the cache for all images associated with it.

Examples:
  cosmoflare images variants delete old-variant
  cosmoflare images variants delete old-variant --force`,
	RunE: runImagesVariantsDelete,
}

func init() {
	rootCmd.AddCommand(imagesCmd)

	imagesCmd.AddCommand(imagesUploadCmd)
	imagesCmd.AddCommand(imagesListCmd)
	imagesCmd.AddCommand(imagesGetCmd)
	imagesCmd.AddCommand(imagesDeleteCmd)
	imagesCmd.AddCommand(imagesVariantsCmd)

	imagesVariantsCmd.AddCommand(imagesVariantsListCmd)
	imagesVariantsCmd.AddCommand(imagesVariantsCreateCmd)
	imagesVariantsCmd.AddCommand(imagesVariantsDeleteCmd)

	// Upload flags
	imagesUploadCmd.Flags().StringVar(&imagesURL, "url", "", "Upload image from URL instead of local file")
	imagesUploadCmd.Flags().StringVar(&imagesMetadata, "metadata", "", "Custom metadata as JSON (e.g., '{\"key\":\"value\"}')")
	imagesUploadCmd.Flags().BoolVar(&imagesRequireSignedURL, "require-signed-urls", false, "Require signed URLs for image access")

	// Delete flags
	imagesDeleteCmd.Flags().BoolVar(&imagesForce, "force", false, "Skip confirmation prompt")

	// Variant create flags
	imagesVariantsCreateCmd.Flags().StringVar(&variantFit, "fit", "scale-down", "Resize fit mode (scale-down, contain, cover, crop, pad)")
	imagesVariantsCreateCmd.Flags().IntVar(&variantWidth, "width", 0, "Variant width in pixels")
	imagesVariantsCreateCmd.Flags().IntVar(&variantHeight, "height", 0, "Variant height in pixels")
	imagesVariantsCreateCmd.Flags().StringVar(&variantMetadataMode, "metadata-mode", "none", "Metadata handling (none, keep, copyright)")
	imagesVariantsCreateCmd.Flags().BoolVar(&variantNeverSignedURLs, "never-require-signed-urls", false, "Allow unsigned access to this variant")

	// Variant delete flags
	imagesVariantsDeleteCmd.Flags().BoolVar(&variantForce, "force", false, "Skip confirmation prompt")
}

func getImagesService() (*cosmoflare.ImagesService, error) {
	return cosmoflare.NewImagesServiceFromCreds(AccountID, APIToken)
}

func runImagesUpload(cmd *cobra.Command, args []string) error {
	// URL upload mode
	if imagesURL != "" {
		return runImagesUploadByURL(cmd)
	}

	// File upload mode — requires a file argument
	if len(args) == 0 {
		return fmt.Errorf("file path is required (or use --url for URL upload)")
	}
	filePath := args[0]

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would upload image", map[string]interface{}{
				"file": filePath,
			})
		}
		printInfo("DRY RUN: Would upload image from '%s'", filePath)
		return nil
	}

	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	// file is closed by the cloudflare-go library after upload

	var metadata map[string]interface{}
	if imagesMetadata != "" {
		if err := json.Unmarshal([]byte(imagesMetadata), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	img, err := svc.UploadFile(context.Background(), file, filePath, metadata)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to upload image: %v", err))
		}
		return fmt.Errorf("failed to upload image: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Image uploaded successfully", img)
	}

	printSuccess("Image uploaded successfully!")
	printInfo("ID: %s", img.ID)
	printInfo("Filename: %s", img.Filename)
	printInfo("Uploaded: %s", img.Uploaded.Format("2006-01-02 15:04:05"))
	printInfo("Variants: %s", strings.Join(img.Variants, ", "))
	return nil
}

func runImagesUploadByURL(cmd *cobra.Command) error {
	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would upload image from URL", map[string]interface{}{
				"url": imagesURL,
			})
		}
		printInfo("DRY RUN: Would upload image from URL '%s'", imagesURL)
		return nil
	}

	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	var opts []cosmoflare.ImageUploadOption
	if imagesRequireSignedURL {
		opts = append(opts, cosmoflare.WithRequireSignedURLs(true))
	}
	if imagesMetadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(imagesMetadata), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
		opts = append(opts, cosmoflare.WithImageMetadata(metadata))
	}

	img, err := svc.UploadByURL(context.Background(), imagesURL, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to upload image from URL: %v", err))
		}
		return fmt.Errorf("failed to upload image from URL: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Image uploaded from URL successfully", img)
	}

	printSuccess("Image uploaded from URL successfully!")
	printInfo("ID: %s", img.ID)
	printInfo("Filename: %s", img.Filename)
	printInfo("Uploaded: %s", img.Uploaded.Format("2006-01-02 15:04:05"))
	printInfo("Variants: %s", strings.Join(img.Variants, ", "))
	return nil
}

func runImagesList(cmd *cobra.Command, args []string) error {
	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	images, err := svc.ListImages(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list images: %v", err))
		}
		return fmt.Errorf("failed to list images: %w", err)
	}

	if JSONOutput {
		return printJSON(images)
	}

	if len(images) == 0 {
		printInfo("No images found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFILENAME\tSIGNED\tVARIANTS\tUPLOADED")
	for _, img := range images {
		signedStr := "no"
		if img.RequireSignedURLs {
			signedStr = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			img.ID,
			img.Filename,
			signedStr,
			len(img.Variants),
			img.Uploaded.Format("2006-01-02 15:04:05"),
		)
	}
	w.Flush()

	printInfo("Total: %d image(s)", len(images))
	return nil
}

func runImagesGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image ID is required")
	}
	imageID := args[0]

	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	img, err := svc.GetImage(context.Background(), imageID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get image: %v", err))
		}
		return fmt.Errorf("failed to get image: %w", err)
	}

	if JSONOutput {
		return printJSON(img)
	}

	fmt.Printf("ID:                %s\n", img.ID)
	fmt.Printf("Filename:          %s\n", img.Filename)
	fmt.Printf("Uploaded:          %s\n", img.Uploaded.Format("2006-01-02 15:04:05"))
	fmt.Printf("Require Signed:    %v\n", img.RequireSignedURLs)
	fmt.Printf("Variants:          %s\n", strings.Join(img.Variants, ", "))
	if len(img.Meta) > 0 {
		metaJSON, _ := json.MarshalIndent(img.Meta, "                   ", "  ")
		fmt.Printf("Metadata:          %s\n", string(metaJSON))
	}
	return nil
}

func runImagesDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("image ID is required")
	}
	imageID := args[0]

	if !imagesForce && !DryRun {
		fmt.Printf("Are you sure you want to delete image '%s'? [y/N]: ", imageID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Image deletion cancelled")
			return nil
		}
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete image", map[string]string{
				"image_id": imageID,
			})
		}
		printInfo("DRY RUN: Would delete image '%s'", imageID)
		return nil
	}

	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	if err := svc.DeleteImage(context.Background(), imageID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete image: %v", err))
		}
		return fmt.Errorf("failed to delete image: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Image deleted successfully", map[string]string{
			"image_id": imageID,
		})
	}
	printSuccess("Image '%s' deleted successfully!", imageID)
	return nil
}

func runImagesVariantsList(cmd *cobra.Command, args []string) error {
	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	variants, err := svc.ListVariants(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list variants: %v", err))
		}
		return fmt.Errorf("failed to list variants: %w", err)
	}

	if JSONOutput {
		return printJSON(variants)
	}

	if len(variants) == 0 {
		printInfo("No variants found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tFIT\tWIDTH\tHEIGHT\tMETADATA")
	for _, v := range variants {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
			v.ID,
			v.Options.Fit,
			v.Options.Width,
			v.Options.Height,
			v.Options.Metadata,
		)
	}
	w.Flush()

	printInfo("Total: %d variant(s)", len(variants))
	return nil
}

func runImagesVariantsCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("variant name is required")
	}
	name := args[0]

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create variant", map[string]interface{}{
				"name":   name,
				"fit":    variantFit,
				"width":  variantWidth,
				"height": variantHeight,
			})
		}
		printInfo("DRY RUN: Would create variant '%s' (%s %dx%d)", name, variantFit, variantWidth, variantHeight)
		return nil
	}

	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	variant, err := svc.CreateVariant(context.Background(), name, variantFit, variantWidth, variantHeight, variantMetadataMode)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create variant: %v", err))
		}
		return fmt.Errorf("failed to create variant: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Variant created successfully", variant)
	}

	printSuccess("Variant '%s' created successfully!", name)
	printInfo("Fit: %s", variant.Options.Fit)
	printInfo("Width: %d", variant.Options.Width)
	printInfo("Height: %d", variant.Options.Height)
	if variant.Options.Metadata != "" {
		printInfo("Metadata: %s", variant.Options.Metadata)
	}
	return nil
}

func runImagesVariantsDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("variant name is required")
	}
	name := args[0]

	if !variantForce && !DryRun {
		fmt.Printf("Are you sure you want to delete variant '%s'? This purges the cache for all associated images. [y/N]: ", name)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Variant deletion cancelled")
			return nil
		}
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete variant", map[string]string{
				"variant": name,
			})
		}
		printInfo("DRY RUN: Would delete variant '%s'", name)
		return nil
	}

	svc, err := getImagesService()
	if err != nil {
		return fmt.Errorf("failed to create Images service: %w", err)
	}

	if err := svc.DeleteVariant(context.Background(), name); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete variant: %v", err))
		}
		return fmt.Errorf("failed to delete variant: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Variant deleted successfully", map[string]string{
			"variant": name,
		})
	}
	printSuccess("Variant '%s' deleted successfully!", name)
	return nil
}
