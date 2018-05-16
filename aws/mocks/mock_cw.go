package mocks

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/coinbase/odin/aws"
)

// CWClient struct
type CWClient struct {
	aws.CWAPI
}

// DeleteAlarms returns
func (m *CWClient) DeleteAlarms(input *cloudwatch.DeleteAlarmsInput) (*cloudwatch.DeleteAlarmsOutput, error) {
	return nil, nil
}

// PutMetricAlarm returns
func (m *CWClient) PutMetricAlarm(input *cloudwatch.PutMetricAlarmInput) (*cloudwatch.PutMetricAlarmOutput, error) {
	return nil, nil
}
