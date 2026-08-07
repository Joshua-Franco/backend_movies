package services

import (
	"context"
	"fmt"
	"strings"

	"backend_movies/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	Client    *s3.Client
	Bucket    string
	PublicURL string
}

func NewR2Client(cfg *config.Config) *R2Client {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.R2AccessKey,
			cfg.R2SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID))
	})

	return &R2Client{
		Client:    client,
		Bucket:    cfg.R2Bucket,
		PublicURL: strings.TrimRight(cfg.R2PublicURL, "/"),
	}
}

type StreamInfo struct {
	TMDBID    string   `json:"tmdb_id"`
	Available bool     `json:"available"`
	MovieName string   `json:"movie_name,omitempty"`
	Qualities []string `json:"qualities,omitempty"`
}

func (r *R2Client) GetStreamInfo(tmdbID int) (*StreamInfo, error) {
	ctx := context.Background()

	resp, err := r.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(r.Bucket),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, err
	}

	for _, obj := range resp.CommonPrefixes {
		folder := *obj.Prefix
		folder = strings.TrimSuffix(folder, "/")

		masterKey := fmt.Sprintf("%s/master.m3u8", folder)
		_, err := r.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(r.Bucket),
			Key:    aws.String(masterKey),
		})
		if err != nil {
			continue
		}

		qualities := r.getQualities(ctx, folder)

		return &StreamInfo{
			TMDBID:    fmt.Sprintf("%d", tmdbID),
			Available: true,
			MovieName: folder,
			Qualities: qualities,
		}, nil
	}

	return &StreamInfo{
		TMDBID:    fmt.Sprintf("%d", tmdbID),
		Available: false,
	}, nil
}

func (r *R2Client) getQualities(ctx context.Context, movieName string) []string {
	qualities := []string{}
	checks := []string{"1080p", "720p", "480p"}

	for _, q := range checks {
		key := fmt.Sprintf("%s/hls_%s/playlist.m3u8", movieName, q)
		_, err := r.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(r.Bucket),
			Key:    aws.String(key),
		})
		if err == nil {
			qualities = append(qualities, q)
		}
	}

	return qualities
}

func (r *R2Client) GetHLSURL(movieName, quality, file string) string {
	r2Quality := "hls_" + quality
	return fmt.Sprintf("%s/%s/%s/%s", r.PublicURL, movieName, r2Quality, file)
}

func (r *R2Client) GetMasterURL(movieName string) string {
	return fmt.Sprintf("%s/%s/master.m3u8", r.PublicURL, movieName)
}
