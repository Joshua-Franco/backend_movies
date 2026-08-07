package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"backend_movies/config"
	"backend_movies/logger"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryService struct {
	cld    *cloudinary.Cloudinary
	config *config.Config
}

func NewCloudinaryService(cfg *config.Config) (*CloudinaryService, error) {
	if cfg.CloudinaryCloudName == "" || cfg.CloudinaryAPIKey == "" || cfg.CloudinaryAPISecret == "" {
		return nil, fmt.Errorf("Cloudinary credentials not configured")
	}

	cld, err := cloudinary.NewFromParams(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}

	return &CloudinaryService{
		cld:    cld,
		config: cfg,
	}, nil
}

func (s *CloudinaryService) UploadFolder(localPath, cloudFolder string) error {
	ctx := context.Background()

	entries, err := filepath.Glob(filepath.Join(localPath, "**", "*"))
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	for _, entry := range entries {
		info, err := os.Stat(entry)
		if err != nil || info.IsDir() {
			continue
		}

		relPath, err := filepath.Rel(localPath, entry)
		if err != nil {
			continue
		}

		publicID := filepath.Join(cloudFolder, relPath)
		publicID = publicID[:len(publicID)-len(filepath.Ext(publicID))]

		_, err = s.cld.Upload.Upload(ctx, entry, uploader.UploadParams{
			PublicID:       publicID,
			ResourceType:   "video",
			Folder:         cloudFolder,
			UseFilename:    boolPtr(true),
			UniqueFilename: boolPtr(false),
			Overwrite:      boolPtr(true),
		})
		if err != nil {
			logger.Warn("Failed to upload %s: %v", entry, err)
			return fmt.Errorf("upload failed for %s: %w", entry, err)
		}
		logger.Info("Uploaded to Cloudinary: %s", publicID)
	}

	logger.Success("Folder uploaded to Cloudinary: %s", cloudFolder)
	return nil
}

func (s *CloudinaryService) GetHLSURL(cloudFolder, quality string) string {
	return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/%s/%s/playlist.m3u8",
		s.config.CloudinaryCloudName, cloudFolder, quality)
}

func (s *CloudinaryService) GetSegmentURL(cloudFolder, quality, segment string) string {
	return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/%s/%s/%s",
		s.config.CloudinaryCloudName, cloudFolder, quality, segment)
}

func (s *CloudinaryService) DeleteFolder(cloudFolder string) error {
	ctx := context.Background()
	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     cloudFolder,
		ResourceType: "video",
		Invalidate:   boolPtr(true),
	})
	return err
}

func boolPtr(b bool) *bool {
	return &b
}