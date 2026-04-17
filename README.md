# Content Pipeline

## About the Project

Content Pipeline is a Go service built with Beego for generating SEO-ready property content from CSV uploads. It accepts a multipart job request, expands prompt templates with property data, sends the prompts to Groq, and stores the generated input, raw output, and final artifacts in S3-compatible storage such as MinIO.

The frontend project that interacts with this service is available at:
[content-generation-app](https://github.com/Rai321han/content-generation-app)

## Contents

- [About the Project](#about-the-project)
- [How It Works](#how-it-works)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Project](#running-the-project)
- [API Usage](#api-usage)
- [CSV Format](#csv-format)

## How It Works

1. A client uploads a CSV file and prompt templates to `POST /api/v1/job`.
2. The service validates the request and CSV structure.
3. Prompt placeholders are replaced with property-specific values.
4. Generates SEO optimized title and description content for each property.
5. The service stores generated JSON files in the configured S3 bucket.

## Tech Stack

- Go
- Beego v2
- Groq API via OpenAI-compatible client
- AWS S3 SDK v2
- MinIO for local S3-compatible storage
- Docker Compose for local infrastructure

## Project Structure

```text
content_pipeline/
├── conf/
│   └── app-sample.conf          # Sample application configuration for local setup
├── controllers/
│   └── job.go                   # HTTP controller for job creation requests
├── models/
│   ├── job.go                   # Job-related domain models
│   └── property.go              # Property data model
├── pkg/
│   ├── groq/
│   │   └── client.go            # Groq client initialization
│   └── s3/
│       └── client.go            # S3/MinIO client initialization
├── routers/
│   └── router.go                # API route registration
├── services/
│   ├── job.go                   # End-to-end job processing workflow
│   ├── llm.go                   # Groq prompt execution and SEO cleanup
│   ├── storage.go               # S3 path building and upload logic
│   └── validation.go            # Request and CSV validation
├── docker-compose-sample.yml    # Sample MinIO compose file for local development
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency lock file
├── main.go                      # Application entry point
├── test.csv                     # Example CSV input
└── README.md                    # Project documentation
```

## Installation

### Prerequisites

- Go
- Docker and Docker Compose
- A Groq API key
- An S3-compatible bucket, or MinIO for local development

### Setup

1. Clone the repository.

```bash
    git clone https://github.com/Rai321han/data-pipeline.git
```

2. Change into the project directory.

```bash
    cd data-pipeline
```

3. Install dependencies:

```bash
    go mod download
```

## Configuration

### Application Config

Use the sample config as the starting point for local development.

```bash
cp conf/app-sample.conf conf/app.conf
```

Update the values in `conf/app.conf` as needed:

- `bucket_name`: S3 or MinIO bucket used to store generated files
- `aws_endpoint`: S3-compatible endpoint such as `http://localhost:9000`
- `aws_access_key`: storage access key
- `aws_secret_key`: storage secret key
- `aws_region`: AWS region or a local value such as `us-east-1`
- `groq_api_key`: Groq API key
- `groq_model`: model name used for content generation
- `groq_base_url`: Groq OpenAI-compatible base URL
- `temperature`: model temperature
- `max_output_tokens`: maximum tokens for generated output
- `frontend_url`: URL of the frontend application for CORS configuration

### Docker Compose

For local MinIO setup, create the active compose file from the sample file provided in this repository, `docker-compose-sample.yml`:

```bash
cp docker-compose-sample.yml docker-compose.yml
```

Then start MinIO:

```bash
docker compose up -d
```

MinIO will be available at:

- S3 API: `http://localhost:9000`
- Console: `http://localhost:9001`

If you use MinIO locally, make sure the bucket from `bucket_name` exists before sending jobs.

## Running the Project

Start the API server with:

```bash
go run main.go
```

The application runs on:

```text
http://localhost:8080
```

## API Usage

### Create Job

- Method: `POST`
- Endpoint: `/api/v1/job`
- Content-Type: `multipart/form-data`

Required form fields:

- `title`: title prompt template
- `description`: description prompt template
- `site`: site identifier used in storage paths
- `file`: CSV file

Example:

```bash
curl -X POST http://localhost:8080/api/v1/job \
  -F 'title=Write an SEO title for {PropertyName}' \
  -F 'description=Write an SEO description for {PropertyDescription}' \
  -F 'site=example-site' \
  -F 'file=@test.csv'
```

Success response:
No content, HTTP status `204 No Content`

## CSV Format

The uploaded CSV must contain exactly these headers:

```csv
id,title,description
```

Rules:

- At least one data row is required
- `id` values must be unique
- No field can be empty

Example:

```csv
id,title,description
101,Sample Property,Spacious apartment near the city center
102,Another Property,Modern home with natural light
```
