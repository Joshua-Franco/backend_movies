package services

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"backend_movies/config"
	"backend_movies/database"
	"backend_movies/logger"
)

type Quality struct {
	Name      string
	Scale     string
	VBitrate  string
	ABitrate  string
}

var Qualities = []Quality{
	{Name: "1080p", Scale: "1920:-2", VBitrate: "5000k", ABitrate: "192k"},
	{Name: "720p", Scale: "1280:-2", VBitrate: "2500k", ABitrate: "128k"},
	{Name: "480p", Scale: "854:-2", VBitrate: "1000k", ABitrate: "96k"},
}

type ConverterService struct {
	config      *config.Config
	cloudinary  *CloudinaryService
	uploadStatus map[string]*UploadStatus
}

type UploadStatus struct {
	UploadID     string
	TMDBID       int
	MovieName    string
	Status       string
	CurrentQuality string
	Progress     int
	Error        string
	StartedAt    time.Time
	CompletedAt  *time.Time
}

func NewConverterService(cfg *config.Config, cld *CloudinaryService) *ConverterService {
	return &ConverterService{
		config:       cfg,
		cloudinary:   cld,
		uploadStatus: make(map[string]*UploadStatus),
	}
}

func (s *ConverterService) CreateUploadStatus(uploadID string, tmdbID int, movieName string) *UploadStatus {
	status := &UploadStatus{
		UploadID:  uploadID,
		TMDBID:    tmdbID,
		MovieName: movieName,
		Status:    "starting",
		Progress:  0,
		StartedAt: time.Now(),
	}
	s.uploadStatus[uploadID] = status
	return status
}

func (s *ConverterService) GetUploadStatus(uploadID string) *UploadStatus {
	return s.uploadStatus[uploadID]
}

func (s *ConverterService) ConvertAndUpload(ctx context.Context, inputPath string, tmdbID int, movieName string) error {
	uploadID := fmt.Sprintf("%d-%s", tmdbID, strings.ReplaceAll(movieName, " ", "-"))
	status := s.CreateUploadStatus(uploadID, tmdbID, movieName)

	logger.Info("=== INICIANDO CONVERSIÓN ===")
	logger.Info("TMDB ID: %d", tmdbID)
	logger.Info("Película: %s", movieName)
	logger.Info("Archivo origen: %s", inputPath)

	defer func() {
		now := time.Now()
		status.CompletedAt = &now
		if r := recover(); r != nil {
			status.Status = "error"
			status.Error = fmt.Sprintf("%v", r)
			logger.Error("Conversión falló: %v", r)
		}
	}()

	duration, err := s.getDuration(inputPath)
	if err != nil {
		status.Status = "error"
		status.Error = err.Error()
		logger.Error("No se pudo obtener duración: %v", err)
		return err
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	logger.Info("Duración total: %02d:%02d:%02d", hours, minutes, seconds)

	tmpDir := filepath.Join(os.TempDir(), "movienova-"+uploadID)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		status.Status = "error"
		status.Error = err.Error()
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, q := range Qualities {
		status.CurrentQuality = q.Name
		status.Status = "converting"
		status.Progress = 0

		logger.Info("========== Convirtiendo a %s ==========", q.Name)

		outDir := filepath.Join(tmpDir, q.Name)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			status.Status = "error"
			status.Error = err.Error()
			return err
		}

		playlistPath := filepath.Join(outDir, "playlist.m3u8")

		cmd := exec.CommandContext(ctx, s.config.FFmpegPath,
			"-i", inputPath,
			"-vf", q.Scale,
			"-c:v", "libx264", "-preset", "fast", "-b:v", q.VBitrate,
			"-c:a", "aac", "-b:a", q.ABitrate,
			"-hls_time", "10", "-hls_list_size", "0", "-f", "hls",
			playlistPath,
		)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			status.Status = "error"
			status.Error = err.Error()
			return err
		}

		if err := cmd.Start(); err != nil {
			status.Status = "error"
			status.Error = err.Error()
			return err
		}

		scanner := bufio.NewScanner(stderr)
		timeRegex := regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})`)
		lastPercent := -1

		for scanner.Scan() {
			line := scanner.Text()
			logger.Info("ffmpeg[%s]: %s", q.Name, line)

			if matches := timeRegex.FindStringSubmatch(line); matches != nil {
				h, _ := strconv.Atoi(matches[1])
				m, _ := strconv.Atoi(matches[2])
				s, _ := strconv.Atoi(matches[3])
				currentSeconds := h*3600 + m*60 + s
				totalSeconds := int(duration.Seconds())
				if totalSeconds > 0 {
					percent := int(float64(currentSeconds) / float64(totalSeconds) * 100)
					if percent > 100 {
						percent = 100
					}
					if percent != lastPercent {
						lastPercent = percent
						status.Progress = percent
						logger.Info("Progreso %s: %d%% (%02d:%02d:%02d / %02d:%02d:%02d)",
							q.Name, percent, h, m, s, hours, minutes, seconds)
					}
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			status.Status = "error"
			status.Error = fmt.Sprintf("ffmpeg %s falló: %v", q.Name, err)
			logger.Error("ffmpeg %s falló: %v", q.Name, err)
			return err
		}

		status.Progress = 100
		logger.Success("%s completado ✓", q.Name)
	}

	status.Status = "uploading"
	logger.Info("Subiendo a Cloudinary...")

	cloudFolder := fmt.Sprintf("movienova/movies/%d", tmdbID)
	if err := s.cloudinary.UploadFolder(tmpDir, cloudFolder); err != nil {
		status.Status = "error"
		status.Error = err.Error()
		logger.Error("Error subiendo a Cloudinary: %v", err)
		return err
	}

	logger.Success("Subida a Cloudinary completada ✓")

	status.Status = "saving_db"
	logger.Info("Guardando en base de datos...")

	for _, q := range Qualities {
		if err := database.SaveMediaFile(tmdbID, "movie", movieName, q.Name, cloudFolder); err != nil {
			logger.Warn("Error guardando %s en BD: %v", q.Name, err)
		}
	}

	status.Status = "completed"
	status.Progress = 100
	logger.Success("=== CONVERSIÓN COMPLETADA PARA TMDB ID: %d ===", tmdbID)

	return nil
}

func (s *ConverterService) getDuration(inputPath string) (time.Duration, error) {
	cmd := exec.Command(s.config.FFprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe falló: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	seconds, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parsear duración: %w", err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}