package client

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/sfn/sfniface"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/aws/s3"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/to"
)

// Deploy attempts to deploy release
func Deploy(releaseFile *string) error {
	region, accountID := to.RegionAccount()
	release, err := releaseFromFile(releaseFile, region, accountID)
	if err != nil {
		return err
	}

	deployerARN := to.StepArn(region, accountID, to.Strp("coinbase-odin"))

	return deploy(&aws.ClientsStr{}, release, deployerARN)
}

func kMSKey() *string {
	// TODO: allow customization of the KMS key from the command line utility
	return to.Strp("alias/aws/s3")
}

func deploy(awsc aws.Clients, release *models.Release, deployerARN *string) error {
	// Uploading the Release to S3 to match SHAs
	if err := s3.PutStruct(awsc.S3Client(nil, nil, nil), release.Bucket, release.ReleasePath(), release); err != nil {
		return err
	}

	// Uploading the encrypted Userdata to S3
	if err := s3.PutSecure(awsc.S3Client(nil, nil, nil), release.Bucket, release.UserDataPath(), release.UserData(), kMSKey()); err != nil {
		return err
	}

	exec, err := findOrCreateExec(awsc.SFNClient(nil, nil, nil), deployerARN, release)
	if err != nil {
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
