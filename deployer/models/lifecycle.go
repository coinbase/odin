package models

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/aws/iam"
	"github.com/coinbase/odin/aws/sns"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// LifeCycleHook struct
type LifeCycleHook struct {
	Transistion      *string `json:"transition,omitempty"`
	SNS              *string `json:"sns,omitempty"`
	Role             *string `json:"role,omitempty"`
	HeartbeatTimeout *int64  `json:"heartbeat_timeout,omitempty"`

	RoleARN               *string `json:"role_arn,omitempty"`
	NotificationTargetARN *string `json:"notification_target_arn,omitempty"`
	Name                  *string `json:"name,omitempty"`
}

// ToLifecycleHookSpecification returns Specification
func (lc *LifeCycleHook) ToLifecycleHookSpecification() *autoscaling.LifecycleHookSpecification {
	return &autoscaling.LifecycleHookSpecification{
		LifecycleHookName: lc.Name,
		HeartbeatTimeout:  lc.HeartbeatTimeout,

		LifecycleTransition: lc.Transistion,

		NotificationTargetARN: lc.NotificationTargetARN,
		RoleARN:               lc.RoleARN,
	}
}

// FetchResources validates resources exist
func (lc *LifeCycleHook) FetchResources(iamc aws.IAMAPI, snsc aws.SNSAPI) error {
	err := iam.RoleExists(iamc, lc.Role)
	if err != nil {
		return err
	}

	if lc.SNS != nil {
		if err := sns.TopicExists(snsc, lc.NotificationTargetARN); err != nil {
			return fmt.Errorf("SNS topic does not exist %v", err.Error())
		}
	}

	return nil
}

// SetDefaults assigns default values
func (lc *LifeCycleHook) SetDefaults(region *string, accountID *string, name string) {
	lc.Name = to.Strp(name)

	if lc.Role != nil && lc.RoleARN == nil {
		lc.RoleARN = to.Strp(fmt.Sprintf("arn:aws:iam::%v:role/%v", *accountID, *lc.Role))
	}

	if lc.SNS != nil && lc.NotificationTargetARN == nil {
		lc.NotificationTargetARN = to.Strp(fmt.Sprintf("arn:aws:sns:%v:%v:%v", *region, *accountID, *lc.SNS))
	}
}

// ValidateAttributes validates attributes
func (lc *LifeCycleHook) ValidateAttributes() error {
	// Quick nil check
	if err := lc.ToLifecycleHookSpecification().Validate(); err != nil {
		return err
	}

	if is.EmptyStr(lc.RoleARN) {
		return fmt.Errorf("Lifecycle RoleARN nil")
	}

	if is.EmptyStr(lc.NotificationTargetARN) {
		return fmt.Errorf("Lifecycle NotificationTargetARN nil")
	}

	if *lc.Transistion != "autoscaling:EC2_INSTANCE_LAUNCHING" && *lc.Transistion != "autoscaling:EC2_INSTANCE_TERMINATING" {
		return fmt.Errorf("Transistion must equal either 'autoscaling:EC2_INSTANCE_LAUNCHING' or 'autoscaling:EC2_INSTANCE_TERMINATING'")
	}

	return nil
}
