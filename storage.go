package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cshum/imagor/storage/s3storage"
)

func NewStorage(bucket, region string) *s3storage.S3Storage {
	sess := session.Must(
		session.NewSession(
			&aws.Config{
				Region:      aws.String(region),
				Credentials: credentials.NewEnvCredentials(),
			},
		),
	)
	return s3storage.New(sess, bucket)
}
