package backends

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GCS is the backend for Google Cloud Storage and implements backends.Storage
type GCS struct {
	ExpiresAfter time.Duration
	Bucket       string
	Client       *storage.Client
}

func (s *GCS) bucket() *storage.BucketHandle {
	return s.Client.Bucket(s.Bucket)
}

// NewGCS returns a new GCS storage backend
func NewGCS(client *storage.Client, bucket string, expiresAfter time.Duration) *GCS {
	return &GCS{
		Bucket:       bucket,
		ExpiresAfter: expiresAfter,
		Client:       client,
	}
}

// Exists returns true if HeadObject does not error
func (s *GCS) Exists(key string) bool {
	obj := s.bucket().Object(key)
	attrs, err := obj.Attrs(context.Background())
	if err != nil || attrs == nil {
		if err != storage.ErrObjectNotExist {
			logrus.WithFields(logrus.Fields{
				"key": key,
			}).WithError(err).Error("GCS: Object lookup returned unknown error")
		}
		return false
	}

	return time.Since(attrs.Updated) < s.ExpiresAfter
}

// Put writes a key to GCS
func (s *GCS) Put(key string, contentType string, body io.ReadSeeker) error {
	obj := s.bucket().Object(key)
	objWriter := obj.NewWriter(context.Background())

	objWriter.ContentType = contentType
	objWriter.ACL = []storage.ACLRule{
		{Entity: storage.AllUsers, Role: storage.RoleReader},
	}
	objWriter.CacheControl = "max-age=1800"

	_, err := io.Copy(objWriter, body)
	if err != nil {
		return errors.Wrap(err, "GCS.Writer.Write")
	}

	err = objWriter.Close()
	if err != nil {
		return errors.Wrap(err, "GCS.Writer.Close")
	}

	logrus.WithFields(logrus.Fields{
		"key":          key,
		"bucket":       s.Bucket,
		"content_type": contentType}).Info("Wrote stub to GCS")

	return nil
}
