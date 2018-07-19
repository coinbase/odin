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

	if message := haltFlag(s3c, release.Bucket, release.HaltPath()); message != nil {
		if *message == "" {
			message = to.Strp("Halt File Found")
		}
		return fmt.Errorf(*message)
	}

	return nil
}

// Halt returns
func (release *Release) Halt(s3c aws.S3API, message *string) error {
	return s3.Put(s3c, release.Bucket, release.HaltPath(), message)
}

// RemoveHalt returns
func (release *Release) RemoveHalt(s3c aws.S3API) {
	if err := s3.Delete(s3c, release.Bucket, release.HaltPath()); err != nil {
		// ignore errors
		fmt.Printf("Warning(RemoveHalt) error ignored: %v\n", err.Error())
	}
}

func haltFlag(s3c aws.S3API, bucket *string, haltPath *string) *string {
	output, body, err := s3.GetObject(s3c, bucket, haltPath)

	// If no file or any error return false
	if err != nil {
		return nil
	}

	// check halt was written in last 5 mins, and before a 2 mins in the future
	if !is.WithinTimeFrame(output.LastModified, 5*time.Minute, 2*time.Minute) {
		return nil
	}

	if body == nil {
		return to.Strp("")
	}

	return to.Strp(string(*body))
}
