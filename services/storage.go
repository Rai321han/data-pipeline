package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/minio/minio-go/v7"
)

type StorageService interface {
	UploadJSON(bucket, key string, data any) error
	DownloadJSON(bucket, key string, out any) error
	UploadBytes(bucket, key string, data []byte, contentType string) error
	BuildInputPath(siteURL string) string
	BuildOutputPath(siteURL string) string
	BuildArtifactPath(siteURL, id string) string
}

type storageService struct {
	client *minio.Client
}

func NewStorageService(minioClient *minio.Client) *storageService {
	return &storageService{
		client: minioClient,
	}
}

func (s *storageService) UploadJSON(bucket, key string, data any) error {

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	reader := bytes.NewReader(jsonData)

	_, err = s.client.PutObject(
		context.Background(),
		bucket,
		key,
		reader,
		int64(len(jsonData)),
		minio.PutObjectOptions{
			ContentType: "application/json",
		},
	)

	return err
}

func (s *storageService) DownloadJSON(bucket, key string, out any) error {
	// Implementation for downloading JSON data from MinIO
	return nil
}

func (s *storageService) UploadBytes(bucket, key string, data []byte, contentType string) error {
	// Implementation for uploading byte data to MinIO
	return nil
}

func (s *storageService) BuildInputPath(siteURL string) string {
	return fmt.Sprintf("%s/details/input/input.json", siteURL)
}

func (s *storageService) BuildOutputPath(siteURL string) string {
	return fmt.Sprintf("%s/details/output/output.json", siteURL)
}

func (s *storageService) BuildArtifactPath(siteURL, id string) string {
	// Implementation for building artifact path based on site URL and ID
	return fmt.Sprintf("%s/details/%s.json", siteURL, id)
}
