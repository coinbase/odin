package alarms

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/coinbase/odin/aws"
)

// AlarmInput struct
type AlarmInput struct {
	*cloudwatch.PutMetricAlarmInput
}

// Create the alarm input
func (alarm *AlarmInput) Create(cwc aws.CWAPI) (*cloudwatch.PutMetricAlarmOutput, error) {
	return cwc.PutMetricAlarm(alarm.PutMetricAlarmInput)
}

// SetAlarmDescription takes the alarms values to build a description
func (alarm *AlarmInput) SetAlarmDescription() {
	desc := []string{}

	if alarm.AlarmName != nil {
		desc = append(desc, fmt.Sprintf("Scale-%v", *alarm.AlarmName))
	}

	if alarm.Statistic != nil && alarm.MetricName != nil {
		desc = append(desc, fmt.Sprintf(" if %v %v", *alarm.Statistic, *alarm.MetricName))
	}

	if alarm.ComparisonOperator != nil && alarm.Threshold != nil {
		desc = append(desc, fmt.Sprintf(" is %v %v%v", *alarm.ComparisonOperator, *alarm.Threshold, "%"))
	}

	if alarm.Period != nil && alarm.EvaluationPeriods != nil {
		desc = append(desc, fmt.Sprintf(" for %v seconds %v times in a row", *alarm.Period, *alarm.EvaluationPeriods))
	}

	descstr := strings.Join(desc, "\n")

	alarm.AlarmDescription = &descstr
}
