package lc

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/step/utils/to"
)

func Test_AddBlockDevice(t *testing.T) {
	input := &LaunchConfigInput{&autoscaling.CreateLaunchConfigurationInput{}}

	input.AddBlockDevice(to.Int64p(10), nil, nil)
	input.AddBlockDevice(to.Int64p(10), to.Strp("asd"), nil)
	input.AddBlockDevice(to.Int64p(10), nil, to.Strp("asd"))

}
