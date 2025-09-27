package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	s3storage "github.com/cshum/imagor/storage/s3storage"
)

func NewStorage(bucket, region string, endpoint *string) *s3storage.S3Storage {
	config := aws.Config{
		Region: region,
	}
	usePathStyle := endpoint != nil

	opts := []s3storage.Option{s3storage.WithForcePathStyle(usePathStyle)}
	if endpoint != nil {
		opts = append(opts, s3storage.WithEndpoint(*endpoint))
	}

	return s3storage.New(config, bucket, opts...)
}
