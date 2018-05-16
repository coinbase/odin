package client

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sfn/sfniface"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/aws/s3"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/to"
)

// Deploy attempts to deploy release
func Deploy(fileOrJSON *string) error {
	region, accountID := to.RegionAccount()
	release, err := releaseFromFileOrJSON(fileOrJSON, region, accountID)
	if err != nil {
		return err
	}

	deployerARN := to.StepArn(region, accountID, to.Strp("coinbase-odin"))

	return deploy(&aws.ClientsStr{}, release, deployerARN)
}

func deploy(awsc aws.Clients, release *models.Release, deployerARN *string) error {
	release.ReleaseID = to.TimeUUID("release-")
	release.CreatedAt = to.Timep(time.Now())

	// Uploading the Release to S3 to match SHAs
	if err := s3.PutStruct(awsc.S3Client(nil, nil, nil), release.Bucket, release.ReleasePath(), release); err != nil {
		return err
	}

	exec, err := findOrCreateExec(awsc.SFNClient(nil, nil, nil), deployerARN, release)
	if err != nil {
		return err
	}

	// Uploading the Release to S3 to match SHAs
	if err := s3.PutStruct(awsc.S3Client(nil, nil, nil), release.Bucket, release.ReleasePath(), release); err != nil {
		return err
	}

	// Execute every second
	exec.WaitForExecution(awsc.SFNClient(nil, nil, nil), 1, waiter)
	fmt.Println("")
	return nil
}

func findOrCreateExec(sfnc sfniface.SFNAPI, deployer *string, release *models.Release) (*execution.Execution, error) {
	exec, err := execution.FindExecution(sfnc, deployer, executionPrefix(release))
	if err != nil {
		return nil, err
	}

	if exec != nil {
		return exec, nil
	}

	return execution.StartExecution(sfnc, deployer, executionName(release), release)
}
