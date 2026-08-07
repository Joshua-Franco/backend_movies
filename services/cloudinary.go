package services

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

func (s *CloudinaryService) GenerateSignedUploadURL(publicID string) map[string]interface{} {
	timestamp := time.Now().Unix()
	
	// Parámetros que deben ser firmados (orden alfabético para consistencia)
	// Cloudinary requiere: folder, public_id, timestamp
	paramsToSign := fmt.Sprintf("folder=movienova/originals&public_id=%s&timestamp=%d", publicID, timestamp)
	
	// Generar signature usando HMAC-SHA1 con el API secret
	h := sha1.New()
	h.Write([]byte(paramsToSign + s.config.CloudinaryAPISecret))
	signature := fmt.Sprintf("%x", h.Sum(nil))
	
	return map[string]interface{}{
		"cloud_name":  s.config.CloudinaryCloudName,
		"api_key":     s.config.CloudinaryAPIKey,
		"timestamp":   timestamp,
		"signature":   signature,
		"public_id":   publicID,
		"folder":      "movienova/originals",
		"upload_url":  fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/video/upload", s.config.CloudinaryCloudName),
	}
}

func (s *CloudinaryService) DownloadOriginal(publicID, destDir string) (string, error) {
	url := fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/%s", s.config.CloudinaryCloudName, publicID)
	
	// Crear directorio destino si no existe
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("crear directorio: %w", err)
	}
	
	// Obtener extensión del publicID o usar .mkv por defecto
	ext := filepath.Ext(publicID)
	if ext == "" {
		ext = ".mkv"
	}
	
	destFile := filepath.Join(destDir, fmt.Sprintf("%s%s", filepath.Base(publicID), ext))
	
	logger.Info("Descargando original de Cloudinary: %s", url)
	
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error HTTP: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Cloudinary retornó %d", resp.StatusCode)
	}
	
	out, err := os.Create(destFile)
	if err != nil {
		return "", fmt.Errorf("crear archivo: %w", err)
	}
	defer out.Close()
	
	// Copiar con progreso
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("guardar archivo: %w", err)
	}
	
	logger.Success("Original descargado: %s (%.2f MB)", destFile, float64(written)/1024/1024)
	return destFile, nil
}