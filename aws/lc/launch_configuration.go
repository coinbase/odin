package lc

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"github.com/coinbase/odin/aws"
)

// Teardown deleted launch configuration
func Teardown(asgc aws.ASGAPI, name *string) error {
	_, err := asgc.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: name,
	})

	if err != nil {
		return err
	}

	return nil
}
