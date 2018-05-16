package models

import (
	"fmt"
	"time"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/aws/s3"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

/////////
// Halt
/////////

// IsHalt will try but no guarantees to work
func (release *Release) IsHalt(s3c aws.S3API) error {
	now := time.Now()

	timeout := release.CreatedAt.Add(time.Second * time.Duration(*release.Timeout))

	if now.After(timeout) {
		return fmt.Errorf("Timeout: Halting Service")
	}

	if isHalt(s3c, release.Bucket, release.HaltPath()) {
		return fmt.Errorf("Halt File Found")
	}

	return nil
}

// Halt returns
func (release *Release) Halt(s3c aws.S3API) error {
	return s3.Put(s3c, release.Bucket, release.HaltPath(), to.Strp("halt"))
}

// RemoveHalt returns
func (release *Release) RemoveHalt(s3c aws.S3API) {
	if err := s3.Delete(s3c, release.Bucket, release.HaltPath()); err != nil {
		// ignore errors
		fmt.Printf("Warning(RemoveHalt) error ignored: %v\n", err.Error())
	}
}

func isHalt(s3c aws.S3API, bucket *string, haltPath *string) bool {
	lm, err := s3.GetLastModified(s3c, bucket, haltPath)

	// If no file or any error return false
	if err != nil {
		return false
	}

	if lm == nil {
		return false
	}

	// check halt was written in last 5 mins, and before a 2 mins in the future
	return is.WithinTimeFrame(lm, 5*time.Minute, 2*time.Minute)
}
