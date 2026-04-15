package services

import (
	"bytes"
	"content_pipeline/models"
	"content_pipeline/pkg/minio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
)

type JobService struct{}

func (s *JobService) CreateJob(title, description, siteUrl string, fileBytes []byte) error {
	reader := csv.NewReader(bytes.NewReader(fileBytes))
	if _, err := reader.Read(); err != nil {
		return err
	}

	properties := make([]models.Property, 0, 16)
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		properties = append(properties, models.Property{
			ID:          record[0],
			Title:       record[1],
			Description: record[2],
		})
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	_ = propertiesJSON

	storageService := NewStorageService(minio.MinioClient)

	inputPath := storageService.BuildInputPath(siteUrl)

	data := &models.JobData{
		SiteURL:     siteUrl,
		Title:       title,
		Description: description,
		Properties:  properties,
	}

	err = storageService.UploadJSON("rebrand-content", inputPath, data)

	if err != nil {
		return err
	}
	return nil
}
