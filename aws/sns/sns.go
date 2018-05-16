package sns

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/coinbase/odin/aws"
)

// TopicExists errors if SNS topic doesn't exists
func TopicExists(snsc aws.SNSAPI, topicARN *string) error {
	_, err := snsc.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: topicARN,
	})

	return err
}
