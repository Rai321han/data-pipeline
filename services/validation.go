package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
)

type ValidationService struct {
}

func (v *ValidationService) ValidateJob(title, description, siteUrl string, fileBytes []byte) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}

	if strings.TrimSpace(siteUrl) == "" {
		return errors.New("site_url is required")
	}

	if strings.TrimSpace(description) == "" {
		return errors.New("description is required")
	}

	if len(description) > 1000 {
		return errors.New("description too long")
	}

	reader := csv.NewReader(bytes.NewReader(fileBytes))

	records, err := reader.ReadAll()
	if err != nil {
		return errors.New("invalid csv file")
	}

	header := records[0]

	if len(header) != 3 ||
		header[0] != "id" ||
		header[1] != "title" ||
		header[2] != "description" {
		return errors.New("invalid csv headers")
	}

	if len(records) < 2 {
		return errors.New("csv file must contain at least one data row")
	}

	ids := make(map[string]bool)

	for i := 1; i < len(records); i++ {

		row := records[i]

		if len(row) != 3 {
			return fmt.Errorf("invalid row at line %d", i+1)
		}

		id := strings.TrimSpace(row[0])
		title := strings.TrimSpace(row[1])
		desc := strings.TrimSpace(row[2])

		if id == "" || title == "" || desc == "" {
			return fmt.Errorf("empty field at line %d", i+1)
		}

		if ids[id] {
			return fmt.Errorf("duplicate id: %s", id)
		}

		ids[id] = true
	}

	return nil
}
