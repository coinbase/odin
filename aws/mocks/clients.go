package mocks

import (
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/aws/mocks"
)

// MockClients struct
type MockClients struct {
	S3  *mocks.MockS3Client
	ASG *ASGClient
	ELB *ELBClient
	EC2 *EC2Client
	ALB *ALBClient
	CW  *CWClient
	IAM *IAMClient
	SNS *SNSClient
	SFN *mocks.MockSFNClient
}

// MockAWS mock clients
func MockAWS() *MockClients {
	return &MockClients{
		S3:  &mocks.MockS3Client{},
		ASG: &ASGClient{},
		ELB: &ELBClient{},
		EC2: &EC2Client{},
		ALB: &ALBClient{},
		CW:  &CWClient{},
		IAM: &IAMClient{},
		SNS: &SNSClient{},
		SFN: &mocks.MockSFNClient{},
	}
}

// S3Client returns
func (a *MockClients) S3Client(*string, *string, *string) aws.S3API {
	return a.S3
}

// ASGClient returns
func (a *MockClients) ASGClient(*string, *string, *string) aws.ASGAPI {
	return a.ASG
}

// ELBClient returns
func (a *MockClients) ELBClient(*string, *string, *string) aws.ELBAPI {
	return a.ELB
}

// EC2Client returns
func (a *MockClients) EC2Client(*string, *string, *string) aws.EC2API {
	return a.EC2
}

// ALBClient returns
func (a *MockClients) ALBClient(*string, *string, *string) aws.ALBAPI {
	return a.ALB
}

// CWClient returns
func (a *MockClients) CWClient(*string, *string, *string) aws.CWAPI {
	return a.CW
}

// IAMClient returns
func (a *MockClients) IAMClient(*string, *string, *string) aws.IAMAPI {
	return a.IAM
}

// SNSClient returns
func (a *MockClients) SNSClient(*string, *string, *string) aws.SNSAPI {
	return a.SNS
}

// SFNClient returns
func (a *MockClients) SFNClient(*string, *string, *string) aws.SFNAPI {
	return a.SFN
}
