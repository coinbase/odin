package alarms

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws"
)

// PolicyInput struct
type PolicyInput struct {
	*autoscaling.PutScalingPolicyInput
}

// Create a policy
func (a *PolicyInput) Create(asgc aws.ASGAPI) (*autoscaling.PutScalingPolicyOutput, error) {
	return asgc.PutScalingPolicy(a.PutScalingPolicyInput)
}
