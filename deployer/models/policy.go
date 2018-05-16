package models

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/aws/alarms"
	"github.com/coinbase/step/utils/to"
)

const cpuScaleDown = "cpu_scale_down"
const cpuScaleUp = "cpu_scale_up"

// Policy struct
type Policy struct {
	serviceID *string

	Type                 *string  `json:"type,omitempty"`
	ScalingAdjustmentVal *int64   `json:"scaling_adjustment,omitempty"`
	ThresholdVal         *float64 `json:"threshold,omitempty"`
	PeriodVal            *int64   `json:"period,omitempty"`
	EvaluationPeriodsVal *int64   `json:"evaluation_periods,omitempty"`
	CooldownVal          *int64   `json:"cooldown,omitempty"`
}

// Name returns name
func (a *Policy) Name() *string {
	return to.Strp(fmt.Sprintf("%v-%v", *a.serviceID, *a.Type))
}

// ScalingAdjustment returns up or down adjustment
func (a *Policy) ScalingAdjustment() *int64 {
	if a.ScalingAdjustmentVal != nil {
		return a.ScalingAdjustmentVal
	}

	switch *a.Type {
	case cpuScaleDown:
		return to.Int64p(-1) // default scale down one
	case cpuScaleUp:
		return to.Int64p(1) // default scale up one
	}

	return to.Int64p(1)
}

// Threshold returns threshold
func (a *Policy) Threshold() *float64 {
	if a.ThresholdVal != nil {
		return a.ThresholdVal
	}

	switch *a.Type {
	case cpuScaleDown:
		return to.Float64p(20)
	case cpuScaleUp:
		return to.Float64p(50)
	}

	return to.Float64p(50)
}

// Period returns period
func (a *Policy) Period() *int64 {
	if a.PeriodVal != nil {
		return a.PeriodVal
	}
	return to.Int64p(300)
}

// EvaluationPeriods returns eval periods
func (a *Policy) EvaluationPeriods() *int64 {
	if a.EvaluationPeriodsVal != nil {
		return a.EvaluationPeriodsVal
	}

	return to.Int64p(2)
}

// Cooldown returns cooldown
func (a *Policy) Cooldown() *int64 {
	if a.CooldownVal != nil {
		return a.CooldownVal
	}
	return to.Int64p(60)
}

// Create attempts to create alarm and policy
func (a *Policy) Create(asgc aws.ASGAPI, cwc aws.CWAPI, asgName *string) error {

	policyInput := a.createPutScalingPolicyInput(asgName)
	output, err := policyInput.Create(asgc)
	if err != nil {
		return err
	}

	alarmInput := a.createMetricAlarmInput(asgName, output.PolicyARN)
	_, err = alarmInput.Create(cwc)

	if err != nil {
		return err
	}

	return nil
}

// ValidateAttributes validates attributes
func (a *Policy) ValidateAttributes() error {
	if a.Type == nil {
		return fmt.Errorf("Policy(?): Type nil")
	}

	if *a.Type != cpuScaleDown && *a.Type != cpuScaleUp {
		return fmt.Errorf("Policy(%v): Unsupported Type %v", *a.Name(), *a.Type)
	}

	if err := a.createMetricAlarmInput(to.Strp("asgName"), nil).Validate(); err != nil {
		return fmt.Errorf("Policy(%v): %v", *a.Name(), err.Error())
	}

	if err := a.createPutScalingPolicyInput(to.Strp("asgName")).Validate(); err != nil {
		return fmt.Errorf("Policy(%v): %v", *a.Name(), err.Error())
	}

	return nil
}

// SetDefaults assigns default values
func (a *Policy) SetDefaults(serviceID *string) error {
	a.serviceID = serviceID
	return nil
}

func (a *Policy) createMetricAlarmInput(asgName *string, policyARN *string) *alarms.AlarmInput {
	alarm := &alarms.AlarmInput{&cloudwatch.PutMetricAlarmInput{}}
	alarm.MetricName = to.Strp("CPUUtilization")
	alarm.Namespace = to.Strp("AWS/EC2")
	alarm.Statistic = to.Strp("Average")
	alarm.ActionsEnabled = to.Boolp(true)
	alarm.Period = a.Period()
	alarm.EvaluationPeriods = a.EvaluationPeriods()
	alarm.AlarmName = a.Name()
	alarm.Threshold = a.Threshold()
	alarm.Dimensions = []*cloudwatch.Dimension{
		&cloudwatch.Dimension{Name: to.Strp("AutoScalingGroupName"), Value: asgName},
	}

	if policyARN != nil {
		alarm.AlarmActions = []*string{policyARN}
	}

	switch *a.Type {
	case cpuScaleUp:
		alarm.ComparisonOperator = to.Strp("GreaterThanThreshold")
	case cpuScaleDown:
		alarm.ComparisonOperator = to.Strp("LessThanThreshold")
	}

	alarm.SetAlarmDescription()

	return alarm
}

func (a *Policy) createPutScalingPolicyInput(asgName *string) *alarms.PolicyInput {
	return &alarms.PolicyInput{&autoscaling.PutScalingPolicyInput{
		AutoScalingGroupName: asgName,
		PolicyName:           a.Type,
		ScalingAdjustment:    a.ScalingAdjustment(),
		AdjustmentType:       to.Strp("ChangeInCapacity"),
		Cooldown:             a.Cooldown(),
	}}
}
