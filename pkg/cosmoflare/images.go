package cosmoflare

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// CFImage represents a Cloudflare Image.
type CFImage struct {
	ID                string                 `json:"id"`
	Filename          string                 `json:"filename"`
	Meta              map[string]interface{} `json:"meta,omitempty"`
	RequireSignedURLs bool                   `json:"requireSignedURLs"`
	Variants          []string               `json:"variants"`
	Uploaded          time.Time              `json:"uploaded"`
}

// CFImageVariant represents a Cloudflare Images delivery variant.
type CFImageVariant struct {
	ID                     string                  `json:"id"`
	NeverRequireSignedURLs bool                    `json:"neverRequireSignedURLs,omitempty"`
	Options                CFImageVariantOptions   `json:"options"`
}

// CFImageVariantOptions holds variant resize/transform options.
type CFImageVariantOptions struct {
	Fit      string `json:"fit"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Metadata string `json:"metadata,omitempty"`
}

// ImagesService implements Cloudflare Images operations.
type ImagesService struct {
	cf        *cloudflare.API
	accountID string
}

// NewImagesService creates a new Images service client.
func NewImagesService(api *cloudflare.API, accountID string) (*ImagesService, error) {
	if api == nil {
		return nil, validationError("NewImagesService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewImagesService", "account ID is required")
	}
	return &ImagesService{cf: api, accountID: accountID}, nil
}

// NewImagesServiceFromCreds creates an ImagesService from account ID and API token.
func NewImagesServiceFromCreds(accountID, apiToken string) (*ImagesService, error) {
	if accountID == "" {
		return nil, validationError("NewImagesService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewImagesService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewImagesService", "failed to create Cloudflare API client", err)
	}
	return &ImagesService{cf: cf, accountID: accountID}, nil
}

// UploadFile uploads an image from a local file.
func (s *ImagesService) UploadFile(ctx context.Context, file io.ReadCloser, filename string, metadata map[string]interface{}) (*CFImage, error) {
	if file == nil {
		return nil, validationError("ImagesService.UploadFile", "file is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.UploadImageParams{
		File:     file,
		Name:     filename,
		Metadata: metadata,
	}

	img, err := s.cf.UploadImage(ctx, rc, params)
	if err != nil {
		return nil, newError("ImagesService.UploadFile", "failed to upload image", err)
	}

	return toCFImage(img), nil
}

// UploadByURL uploads an image from a remote URL.
func (s *ImagesService) UploadByURL(ctx context.Context, url string, opts ...ImageUploadOption) (*CFImage, error) {
	if url == "" {
		return nil, validationError("ImagesService.UploadByURL", "URL is required")
	}

	cfg := &imageUploadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.UploadImageParams{
		URL:               url,
		RequireSignedURLs: cfg.requireSignedURLs,
		Metadata:          cfg.metadata,
	}

	img, err := s.cf.UploadImage(ctx, rc, params)
	if err != nil {
		return nil, newError("ImagesService.UploadByURL", "failed to upload image from URL", err)
	}

	return toCFImage(img), nil
}

// ListImages returns all images in the account.
func (s *ImagesService) ListImages(ctx context.Context) ([]*CFImage, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, err := s.cf.ListImages(ctx, rc, cloudflare.ListImagesParams{})
	if err != nil {
		return nil, newError("ImagesService.ListImages", "failed to list images", err)
	}

	images := make([]*CFImage, 0, len(results))
	for _, img := range results {
		images = append(images, toCFImage(img))
	}
	return images, nil
}

// GetImage retrieves a single image by ID.
func (s *ImagesService) GetImage(ctx context.Context, imageID string) (*CFImage, error) {
	if imageID == "" {
		return nil, validationError("ImagesService.GetImage", "image ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	img, err := s.cf.GetImage(ctx, rc, imageID)
	if err != nil {
		return nil, notFound("ImagesService.GetImage", "", imageID, err)
	}

	return toCFImage(img), nil
}

// DeleteImage deletes an image by ID.
func (s *ImagesService) DeleteImage(ctx context.Context, imageID string) error {
	if imageID == "" {
		return validationError("ImagesService.DeleteImage", "image ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DeleteImage(ctx, rc, imageID); err != nil {
		return newError("ImagesService.DeleteImage", fmt.Sprintf("failed to delete image %q", imageID), err)
	}
	return nil
}

// ListVariants returns all image delivery variants.
func (s *ImagesService) ListVariants(ctx context.Context) ([]*CFImageVariant, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.ListImagesVariants(ctx, rc, cloudflare.ListImageVariantsParams{})
	if err != nil {
		return nil, newError("ImagesService.ListVariants", "failed to list variants", err)
	}

	variants := make([]*CFImageVariant, 0, len(result.ImagesVariants))
	for _, v := range result.ImagesVariants {
		variants = append(variants, toCFImageVariant(v))
	}
	return variants, nil
}

// CreateVariant creates a new image delivery variant.
func (s *ImagesService) CreateVariant(ctx context.Context, name, fit string, width, height int, metadataMode string) (*CFImageVariant, error) {
	if name == "" {
		return nil, validationError("ImagesService.CreateVariant", "variant name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.CreateImagesVariantParams{
		ID: name,
		Options: cloudflare.ImagesVariantsOptions{
			Fit:      fit,
			Width:    width,
			Height:   height,
			Metadata: metadataMode,
		},
	}

	v, err := s.cf.CreateImagesVariant(ctx, rc, params)
	if err != nil {
		return nil, newError("ImagesService.CreateVariant", fmt.Sprintf("failed to create variant %q", name), err)
	}

	return toCFImageVariant(v), nil
}

// DeleteVariant deletes an image delivery variant.
func (s *ImagesService) DeleteVariant(ctx context.Context, name string) error {
	if name == "" {
		return validationError("ImagesService.DeleteVariant", "variant name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DeleteImagesVariant(ctx, rc, name); err != nil {
		return newError("ImagesService.DeleteVariant", fmt.Sprintf("failed to delete variant %q", name), err)
	}
	return nil
}

// --- Image upload options ---

// ImageUploadOption is a functional option for image upload operations.
type ImageUploadOption func(*imageUploadConfig)

type imageUploadConfig struct {
	requireSignedURLs bool
	metadata          map[string]interface{}
}

// WithRequireSignedURLs sets whether the image requires signed URLs for access.
func WithRequireSignedURLs(v bool) ImageUploadOption {
	return func(c *imageUploadConfig) { c.requireSignedURLs = v }
}

// WithImageMetadata attaches custom metadata to an uploaded image.
func WithImageMetadata(m map[string]interface{}) ImageUploadOption {
	return func(c *imageUploadConfig) { c.metadata = m }
}

// --- Conversion helpers ---

func toCFImage(img cloudflare.Image) *CFImage {
	return &CFImage{
		ID:                img.ID,
		Filename:          img.Filename,
		Meta:              img.Meta,
		RequireSignedURLs: img.RequireSignedURLs,
		Variants:          img.Variants,
		Uploaded:          img.Uploaded,
	}
}

func toCFImageVariant(v cloudflare.ImagesVariant) *CFImageVariant {
	neverSigned := false
	if v.NeverRequireSignedURLs != nil {
		neverSigned = *v.NeverRequireSignedURLs
	}
	return &CFImageVariant{
		ID:                     v.ID,
		NeverRequireSignedURLs: neverSigned,
		Options: CFImageVariantOptions{
			Fit:      v.Options.Fit,
			Width:    v.Options.Width,
			Height:   v.Options.Height,
			Metadata: v.Options.Metadata,
		},
	}
}
