package services

import (
	"os"

	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

func UploadPublicFile(path string, data []byte, contType string) string {
	s3Region := aws.USEast
	s3BucketName := os.Getenv("AWS_S3_BUCKET_NAME")

	auth, err := aws.EnvAuth()

	if err != nil {
		panic(err.Error())
	}

	// Open Bucket
	s3Client := s3.New(auth, s3Region)
	s3Bucket := s3Client.Bucket(s3BucketName)

	err = s3Bucket.Put(path, data, contType, s3.PublicRead)

	if err != nil {
		panic(err.Error())
	}

	return s3Region.S3Endpoint + "/" + s3BucketName + "/" + path
}
