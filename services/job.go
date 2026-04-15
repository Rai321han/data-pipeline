package services

import (
	"bytes"
	"content_pipeline/models"
	"content_pipeline/pkg/geminiai"
	"content_pipeline/pkg/minio"
	"encoding/csv"
	"errors"
	"io"
	"sync"

	beego "github.com/beego/beego/v2/server/web"
)

type JobService struct{}

func (s *JobService) ProcessJob(title, description, siteUrl string, fileBytes []byte) (string, error) {
	bucket, err := beego.AppConfig.String("app::bucket_name")

	if err != nil || bucket == "" {
		return "", errors.New("bucket name is not configured")
	}

	reader := csv.NewReader(bytes.NewReader(fileBytes))
	if _, err := reader.Read(); err != nil {
		return "", err
	}

	properties := []string{}
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}

		properties = append(properties, record[0])
	}

	storageService := NewStorageService(minio.MinioClient)

	inputPath := storageService.BuildInputPath(siteUrl)

	data := &models.JobData{
		SiteURL:     siteUrl,
		Title:       title,
		Description: description,
		Properties:  properties,
	}

	err = storageService.UploadJSON(bucket, inputPath, data)

	if err != nil {
		return "", err
	}

	model := beego.AppConfig.DefaultString("app::model", "gemini-2.5-flash-lite")
	llm := NewLLMService(geminiai.Genaiclient, model)

	seoResponse, err := llm.GenerateSEO(title, description)

	if err != nil {
		return "", err
	}

	outputPath := storageService.BuildOutputPath(siteUrl)

	err = storageService.UploadJSON(bucket, outputPath, seoResponse)

	if err != nil {
		return "", err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(properties))

	for _, propertyID := range properties {
		wg.Add(1)

		go func(propertyID string) {
			defer wg.Done()

			artifactPath := storageService.BuildArtifactPath(siteUrl, propertyID)
			artifactData := map[string]string{
				"id":          propertyID,
				"title":       seoResponse.Title,
				"description": seoResponse.Description,
			}

			if err := storageService.UploadJSON(bucket, artifactPath, artifactData); err != nil {
				errCh <- err
			}
		}(propertyID)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return "", err
		}
	}

	return inputPath, nil
}
