package models

import (
	"github.com/coinbase/step/aws"
	"github.com/coinbase/step/aws/s3"
)

// GrabLock tries to grab the lock
func (release *Release) GrabLock(s3Client aws.S3API) (bool, error) {
	return s3.GrabLock(s3Client, release.Bucket, release.LockPath(), *release.UUID)
}

// ReleaseLock tries to release the lock
func (release *Release) ReleaseLock(s3Client aws.S3API) error {
	return s3.ReleaseLock(s3Client, release.Bucket, release.LockPath(), *release.UUID)
}
