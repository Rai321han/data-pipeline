package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

type ValidationService struct {
}

// ValidateJob validates the input parameters for a job, including the title, description, site URL, and the contents of the CSV file.
// It checks for the presence of required fields, validates the length of the description, and ensures that the CSV file has the correct format and contains valid data.
// Parameters:
//
//   - title: The title of the job, which must be a non-empty string.
//
//   - description: A description of the job, which must be a non-empty string and not exceed 1000 characters in length.
//
//   - siteUrl: The URL of the site associated with the job, which must be a non-empty string.
//
//   - fileBytes: The contents of the CSV file as a byte slice. The CSV file must have a header row with the columns "id", "title", and "description", and at least one data row. Each data row must have non-empty values for all three columns, and the "id" values must be unique.
//
// Returns an error object if any validation checks fail, or nil if all validations pass successfully.
func (v *ValidationService) ValidateJob(title, description, siteUrl string, fileBytes []byte) error {
	if err := v.validateRequiredFields(title, description, siteUrl); err != nil {
		return err
	}

	records, err := v.readCSVRecords(fileBytes)
	if err != nil {
		return err
	}

	if err := v.validateCSVStructure(records); err != nil {
		return err
	}

	return v.validateDataRows(records)
}

func (v *ValidationService) validateRequiredFields(title, description, siteURL string) error {
	if strings.TrimSpace(title) == "" {
		return NewServiceError(ErrValidation, "TITLE_REQUIRED", "title is required", nil)
	}

	if strings.TrimSpace(siteURL) == "" {
		return NewServiceError(ErrValidation, "SITE_URL_REQUIRED", "site_url is required", nil)
	}

	if strings.TrimSpace(description) == "" {
		return NewServiceError(ErrValidation, "DESCRIPTION_REQUIRED", "description is required", nil)
	}

	return nil
}

func (v *ValidationService) readCSVRecords(fileBytes []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(fileBytes))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, NewServiceError(ErrValidation, "INVALID_CSV", "invalid csv file", err)
	}

	return records, nil
}

func (v *ValidationService) validateCSVStructure(records [][]string) error {
	if len(records) == 0 {
		return NewServiceError(ErrValidation, "INVALID_CSV", "csv file is empty", nil)
	}

	header := records[0]
	if len(header) != 3 || header[0] != "id" || header[1] != "title" || header[2] != "description" {
		return NewServiceError(ErrValidation, "INVALID_CSV_HEADERS", "invalid csv headers", nil)
	}

	if len(records) < 2 {
		return NewServiceError(ErrValidation, "CSV_DATA_REQUIRED", "csv file must contain at least one data row", nil)
	}

	return nil
}

func (v *ValidationService) validateDataRows(records [][]string) error {
	ids := make(map[string]bool)

	for i := 1; i < len(records); i++ {
		if err := v.validateSingleRow(records[i], i+1, ids); err != nil {
			return err
		}
	}

	return nil
}

func (v *ValidationService) validateSingleRow(row []string, line int, ids map[string]bool) error {
	if len(row) != 3 {
		return NewServiceError(ErrValidation, "INVALID_CSV_ROW", fmt.Sprintf("invalid row at line %d", line), nil)
	}

	id := strings.TrimSpace(row[0])
	rowTitle := strings.TrimSpace(row[1])
	rowDescription := strings.TrimSpace(row[2])

	if id == "" || rowTitle == "" || rowDescription == "" {
		return NewServiceError(ErrValidation, "EMPTY_CSV_FIELD", fmt.Sprintf("empty field at line %d", line), nil)
	}

	if ids[id] {
		return NewServiceError(ErrValidation, "DUPLICATE_ID", fmt.Sprintf("duplicate id: %s", id), nil)
	}

	ids[id] = true
	return nil
}
