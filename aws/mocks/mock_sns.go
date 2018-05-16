package mocks

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/coinbase/odin/aws"
)

// SNSClient returns
type SNSClient struct {
	aws.SNSAPI
}

// GetTopicAttributes returns
func (m *SNSClient) GetTopicAttributes(in *sns.GetTopicAttributesInput) (*sns.GetTopicAttributesOutput, error) {
	return nil, nil
}
