package lc

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

var ebsOptimizedInstances = map[string]bool{
	"c4.large":    true,
	"c4.xlarge":   true,
	"c4.2xlarge":  true,
	"c4.4xlarge":  true,
	"c4.8xlarge":  true,
	"c5.large":    true,
	"c5.xlarge":   true,
	"c5.2xlarge":  true,
	"c5.4xlarge":  true,
	"c5.9xlarge":  true,
	"c5.18xlarge": true,
	"i3.large":    true,
	"i3.xlarge":   true,
	"i3.2xlarge":  true,
	"i3.4xlarge":  true,
	"i3.8xlarge":  true,
	"i3.16xlarge": true,
	"m4.large":    true,
	"m4.xlarge":   true,
	"m4.2xlarge":  true,
	"m4.4xlarge":  true,
	"m4.10xlarge": true,
	"m4.16xlarge": true,
	"m5.large":    true,
	"m5.xlarge":   true,
	"m5.2xlarge":  true,
	"m5.4xlarge":  true,
	"m5.12xlarge": true,
	"m5.24xlarge": true,
	"r4.large":    true,
	"r4.xlarge":   true,
	"r4.2xlarge":  true,
	"r4.4xlarge":  true,
	"r4.8xlarge":  true,
	"r4.16xlarge": true,
}

// LaunchConfigInput input struct
type LaunchConfigInput struct {
	*autoscaling.CreateLaunchConfigurationInput
}

// Create tryes to create the launch configuration
func (s *LaunchConfigInput) Create(asgc aws.ASGAPI) error {
	if err := s.Validate(); err != nil {
		return err
	}

	_, err := asgc.CreateLaunchConfiguration(s.CreateLaunchConfigurationInput)

	if err != nil {
		return err
	}

	return nil
}

// AddBlockDevice adds an EBS block device to the LC
func (s *LaunchConfigInput) AddBlockDevice(ebsVolumeSize *int64, ebsVolumeType *string, ebsDeviceType *string) {
	if ebsVolumeSize == nil {
		return
	}

	if ebsVolumeType == nil {
		ebsVolumeType = to.Strp("gp2")
	}

	if ebsDeviceType == nil {
		ebsDeviceType = to.Strp("/dev/xvda")
	}

	block := &autoscaling.BlockDeviceMapping{
		DeviceName: ebsDeviceType,
		Ebs: &autoscaling.Ebs{
			VolumeSize: ebsVolumeSize,
			VolumeType: ebsVolumeType,
		},
	}

	if s.BlockDeviceMappings == nil {
		s.BlockDeviceMappings = []*autoscaling.BlockDeviceMapping{}
	}

	s.BlockDeviceMappings = append(s.BlockDeviceMappings, block)
}

// SetDefaults assigns values
func (s *LaunchConfigInput) SetDefaults() {
	if s.InstanceType == nil {
		s.InstanceType = to.Strp("t2.nano")
	}

	if s.InstanceMonitoring == nil {
		s.InstanceMonitoring = &autoscaling.InstanceMonitoring{Enabled: to.Boolp(false)}
	}

	if s.EbsOptimized == nil {
		opt := ebsOptimizedInstances[*s.InstanceType]
		s.EbsOptimized = to.Boolp(opt)
	}
}
