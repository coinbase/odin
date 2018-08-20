package client

import (
	"fmt"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/to"
)

// Halt attempts to halt release
func Halt(step_fn *string, releaseFile *string) error {
	region, accountID := to.RegionAccount()
	release, err := releaseFromFile(releaseFile, region, accountID)
	if err != nil {
		return err
	}

	deployerARN := to.StepArn(region, accountID, step_fn)

	return halt(&aws.ClientsStr{}, release, deployerARN)
}

func halt(awsc aws.Clients, release *models.Release, deployerARN *string) error {
	exec, err := execution.FindExecution(awsc.SFNClient(nil, nil, nil), deployerARN, release.ExecutionPrefix())
	if err != nil {
		return err
	}

	if exec == nil {
		return fmt.Errorf("Cannot find current execution of release with prefix %q", release.ExecutionPrefix())
	}

	if err := release.Halt(awsc.S3Client(nil, nil, nil), to.Strp("Odin client Halted deploy")); err != nil {
		return err
	}

	exec.WaitForExecution(awsc.SFNClient(nil, nil, nil), 1, waiter)
	fmt.Println("")
	return nil
}
