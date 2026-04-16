package services

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageService defines the interface for interacting with the storage layer minio/aws s3.
type StorageService interface {
	UploadJSON(bucket, key string, data any) error
	DownloadJSON(bucket, key string, out any) error
	UploadBytes(bucket, key string, data []byte, contentType string) error
	BuildInputPath(siteURL string) string
	BuildOutputPath(siteURL string) string
	BuildArtifactPath(siteURL, id string) string
}

// storageService is a concrete implementation of the StorageService interface that uses a MinIO client to interact with the storage layer.
type storageService struct {
	client *s3.Client
}

func NewStorageService(S3Client *s3.Client) *storageService {
	return &storageService{
		client: S3Client,
	}
}

// UploadJSON uploads a JSON object to the specified bucket and key in S3.
// It marshals the provided data into JSON format and uploads it to the storage service.
// Parameters:
//
//   - bucket: The name of the S3 bucket where the JSON object will be stored.
//
//   - key: The key (path) under which the JSON object will be stored in the bucket.
//
//   - data: The data to be marshaled into JSON and uploaded. It can be any Go data structure that can be marshaled into JSON.
//
// Returns an error object if the upload fails, or nil if the upload is successful.
func (s *storageService) Upload(ctx context.Context, bucket, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		fmt.Printf("Error uploading to S3: %v\n", err)
		return fmt.Errorf("s3 put object bucket=%s key=%s: %w", bucket, key, err)
	}

	return nil
}

// BuildInputPath  constructs the S3 key for the input JSON file based on the provided site URL.
// The key is formatted as "{siteURL}/details/input/input.json", where {siteURL} is the input parameter.
func (s *storageService) BuildInputPath(siteURL string) string {
	return fmt.Sprintf("%s/details/input/input.json", siteURL)
}

// BuildOutputPath constructs the S3 key for the output JSON file based on the provided site URL.
// The key is formatted as "{siteURL}/details/output/output.json", where {siteURL} is the input parameter.
func (s *storageService) BuildOutputPath(siteURL string) string {
	return fmt.Sprintf("%s/details/output/output.json", siteURL)
}

// BuildArtifactPath constructs the S3 key for an artifact JSON file based on the provided site URL and artifact ID.
// The key is formatted as "{siteURL}/details/{id}.json", where {siteURL} is the input parameter and {id} is the artifact ID.
func (s *storageService) BuildArtifactPath(siteURL, id string) string {
	return fmt.Sprintf("%s/details/%s.json", siteURL, id)
}
