package backends

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

// S3 is the backend for s3 data storage and implements backends.Storage
type S3 struct {
	Bucket string
	Svc    *s3.S3
}

// NewS3 returns a new S3 storage backend
func NewS3(svc *s3.S3, bucket string) *S3 {
	return &S3{
		Bucket: bucket,
		Svc:    svc,
	}
}

// Exists returns true if HeadObject does not error
func (s *S3) Exists(key string) bool {
	_, err := s.Svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

// Put writes a key to S3
func (s *S3) Put(key string, contentType string, body io.ReadSeeker) error {
	putObjectParams := &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        body,
	}
	_, err := s.Svc.PutObject(putObjectParams)
	if err != nil {
		return errors.Wrap(err, "s3.PutObject")
	}
	logrus.WithFields(logrus.Fields{
		"key":          key,
		"bucket":       s.Bucket,
		"content_type": contentType}).Info("Wrote stub to s3")

	return nil
}
