package asg

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// Input input struct
type Input struct {
	*autoscaling.CreateAutoScalingGroupInput
}

// Create calls to create an ASG
func (s *Input) Create(asgc aws.ASGAPI) error {
	if err := s.Validate(); err != nil {
		return err
	}

	_, err := asgc.CreateAutoScalingGroup(s.CreateAutoScalingGroupInput)

	if err != nil {
		return err
	}

	return nil
}

// SetDefaults assigns default values
func (s *Input) SetDefaults() {
	if s.MinSize == nil {
		s.MinSize = to.Int64p(1)
	}

	if s.MaxSize == nil {
		s.MaxSize = to.Int64p(1)
	}

	if s.DesiredCapacity == nil {
		s.DesiredCapacity = to.Int64p(1)
	}

	if s.DefaultCooldown == nil {
		s.DefaultCooldown = to.Int64p(300)
	}

	if s.HealthCheckGracePeriod == nil {
		s.HealthCheckGracePeriod = to.Int64p(300)
	}

	if s.LaunchConfigurationName == nil {
		s.LaunchConfigurationName = s.AutoScalingGroupName // Makes the name the same
	}

	s.HealthCheckType = to.Strp("EC2")
	if len(s.LoadBalancerNames) > 0 || len(s.TargetGroupARNs) > 0 {
		s.HealthCheckType = to.Strp("ELB") // If there are any ELBs set the health check to that
	}

	if len(s.TerminationPolicies) == 0 {
		s.TerminationPolicies = []*string{to.Strp("ClosestToNextInstanceHour")}
	}
}

// AddTag adds a tag to the input
func (s *Input) AddTag(key string, value *string) {
	if s.Tags == nil {
		s.Tags = []*autoscaling.Tag{}
	}

	for _, tag := range s.Tags {
		if *tag.Key == key {
			tag.Value = value
			return // Found the tag key already
		}
	}

	// Add new Tag
	s.Tags = append(s.Tags, &autoscaling.Tag{Key: &key, Value: value, PropagateAtLaunch: to.Boolp(true)})
}

// ToASG returns ASG object
func (s *Input) ToASG() *ASG {
	asg := ASG{}
	asg.AutoScalingGroupName = s.AutoScalingGroupName
	return &asg
}
