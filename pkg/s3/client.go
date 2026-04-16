package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client is a global variable that holds the initialized S3 client for making API calls to the S3 service.
// It is initialized in the Init function with the provided configuration parameters.
var S3Client *s3.Client

// Init initializes the S3Client with the given configuration parameters. It should be called before making any API calls to ensure that the client is properly configured.
// Parameters:
//   - endpoint: The endpoint URL for the S3 service.
//   - accessKey: The access key for authenticating with the S3 service.
//   - secretKey: The secret key for authenticating with the S3 service.
//   - region: The region where the S3 service is located.
//
// This function sets up the S3Client with the specified configuration, allowing it to be used throughout the application for interacting with the S3 service.
func Init(endpoint, accessKey, secretKey, region string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				"",
			),
		),
	)
	if err != nil {
		return err
	}
	S3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return nil
}
