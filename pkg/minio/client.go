package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	Conn *minio.Client
}

var MinioClient *minio.Client

func Init(endpoint, accessKey, secretKey string, useSSL bool) error {

	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		return err
	}

	MinioClient = c
	return nil
}
