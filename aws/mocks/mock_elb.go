package mocks

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// DescribeLoadBalancersResponse returns
type DescribeLoadBalancersResponse struct {
	Resp  *elb.DescribeLoadBalancersOutput
	Error error
}

// DescribeTagsResponse returns
type DescribeTagsResponse struct {
	Resp  *elb.DescribeTagsOutput
	Error error
}

// DescribeInstanceHealthResponse returns
type DescribeInstanceHealthResponse struct {
	Resp  *elb.DescribeInstanceHealthOutput
	Error error
}

// ELBClient returns
type ELBClient struct {
	aws.ELBAPI
	DescribeLoadBalancersResp  map[string]*DescribeLoadBalancersResponse
	DescribeTagsResp           map[string]*DescribeTagsResponse
	DescribeInstanceHealthResp map[string]*DescribeInstanceHealthResponse
}

// AWSELBNotFoundError returns
func AWSELBNotFoundError() error {
	return awserr.New(elb.ErrCodeAccessPointNotFoundException, "LoadBalancerNotFound", nil)
}

func (m *ELBClient) init() {
	if m.DescribeLoadBalancersResp == nil {
		m.DescribeLoadBalancersResp = map[string]*DescribeLoadBalancersResponse{}
	}

	if m.DescribeTagsResp == nil {
		m.DescribeTagsResp = map[string]*DescribeTagsResponse{}
	}

	if m.DescribeInstanceHealthResp == nil {
		m.DescribeInstanceHealthResp = map[string]*DescribeInstanceHealthResponse{}
	}
}

// AddELB returns
func (m *ELBClient) AddELB(name string, projectName string, configName string, serviceName string) {
	m.init()
	m.DescribeLoadBalancersResp[name] = &DescribeLoadBalancersResponse{
		Resp: &elb.DescribeLoadBalancersOutput{
			LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
				&elb.LoadBalancerDescription{LoadBalancerName: &name},
			},
		},
	}

	m.DescribeTagsResp[name] = &DescribeTagsResponse{
		Resp: &elb.DescribeTagsOutput{
			TagDescriptions: []*elb.TagDescription{
				&elb.TagDescription{
					LoadBalancerName: &name,
					Tags: []*elb.Tag{
						&elb.Tag{Key: to.Strp("ProjectName"), Value: to.Strp(projectName)},
						&elb.Tag{Key: to.Strp("ConfigName"), Value: to.Strp(configName)},
						&elb.Tag{Key: to.Strp("ServiceName"), Value: to.Strp(serviceName)},
					},
				},
			},
		},
	}

	m.DescribeInstanceHealthResp[name] = &DescribeInstanceHealthResponse{
		Resp: &elb.DescribeInstanceHealthOutput{
			InstanceStates: []*elb.InstanceState{
				&elb.InstanceState{
					InstanceId: to.Strp("InstanceId1"),
					State:      to.Strp("InService"),
				},
			},
		},
	}

}

// DescribeLoadBalancers returns
func (m *ELBClient) DescribeLoadBalancers(in *elb.DescribeLoadBalancersInput) (*elb.DescribeLoadBalancersOutput, error) {
	m.init()
	lbName := in.LoadBalancerNames[0]
	resp := m.DescribeLoadBalancersResp[*lbName]
	if resp == nil {
		return nil, AWSELBNotFoundError()
	}
	return resp.Resp, resp.Error
}

// DescribeTags returns
func (m *ELBClient) DescribeTags(in *elb.DescribeTagsInput) (*elb.DescribeTagsOutput, error) {
	m.init()
	lbName := in.LoadBalancerNames[0]
	resp := m.DescribeTagsResp[*lbName]
	if resp == nil {
		return nil, AWSELBNotFoundError()
	}
	return resp.Resp, resp.Error
}

// DescribeInstanceHealth returns
func (m *ELBClient) DescribeInstanceHealth(in *elb.DescribeInstanceHealthInput) (*elb.DescribeInstanceHealthOutput, error) {
	m.init()
	lbName := in.LoadBalancerName
	resp := m.DescribeInstanceHealthResp[*lbName]
	if resp == nil {
		return nil, AWSELBNotFoundError()
	}
	if resp.Resp == nil {
		return &elb.DescribeInstanceHealthOutput{}, nil
	}
	return resp.Resp, resp.Error
}
